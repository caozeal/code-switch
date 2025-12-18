//go:build darwin
// +build darwin

package services

/*
#cgo darwin LDFLAGS: -framework Carbon

#include <Carbon/Carbon.h>

extern void hotkeyCallbackBridge(UInt32 hotkeyID);

static OSStatus hotkeyHandler(EventHandlerCallRef nextHandler, EventRef event, void *userData) {
	#pragma unused(nextHandler)
	#pragma unused(userData)

	EventHotKeyID hotKeyID;
	OSStatus err = GetEventParameter(
		event,
		kEventParamDirectObject,
		typeEventHotKeyID,
		NULL,
		sizeof(hotKeyID),
		NULL,
		&hotKeyID
	);
	if (err != noErr) {
		return err;
	}
	hotkeyCallbackBridge(hotKeyID.id);
	return noErr;
}

static void setupHandler() {
	static int installed = 0;
	if (installed) {
		return;
	}
	installed = 1;

	EventTypeSpec eventType;
	eventType.eventClass = kEventClassKeyboard;
	eventType.eventKind = kEventHotKeyPressed;

	InstallEventHandler(
		GetApplicationEventTarget(),
		NewEventHandlerUPP(hotkeyHandler),
		1,
		&eventType,
		NULL,
		NULL
	);
}

static OSStatus registerHotkey(UInt32 keycode, UInt32 modifiers, UInt32 id, EventHotKeyRef *outRef) {
	EventHotKeyID hotKeyID;
	hotKeyID.signature = 'CSHK';
	hotKeyID.id = id;
	return RegisterEventHotKey(keycode, modifiers, hotKeyID, GetApplicationEventTarget(), 0, outRef);
}

static OSStatus unregisterHotkey(EventHotKeyRef ref) {
	return UnregisterEventHotKey(ref);
}
*/
import "C"

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
)

const (
	// 前端修饰键 bit 掩码定义
	frontendModifierCmd   uint32 = 1 << 8
	frontendModifierShift uint32 = 1 << 9
	frontendModifierAlt   uint32 = 1 << 10
	frontendModifierCtrl  uint32 = 1 << 11
)

const (
	toggleWindowTarget = "toggle_window"
	hotkeyIDToggle     = 1
)

var (
	hotkeyHandlerOnce sync.Once

	globalHotkeyCallbackMu sync.RWMutex
	globalHotkeyCallback   func(hotkeyID uint32)
)

func setGlobalHotkeyCallback(cb func(hotkeyID uint32)) {
	globalHotkeyCallbackMu.Lock()
	defer globalHotkeyCallbackMu.Unlock()
	globalHotkeyCallback = cb
}

//export hotkeyCallbackBridge
func hotkeyCallbackBridge(hotkeyID C.UInt32) {
	globalHotkeyCallbackMu.RLock()
	cb := globalHotkeyCallback
	globalHotkeyCallbackMu.RUnlock()
	if cb == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("WARN 全局快捷键回调发生 panic: %v", r)
		}
	}()
	cb(uint32(hotkeyID))
}

type HotkeyService struct {
	mu         sync.Mutex
	registered map[string]C.EventHotKeyRef
	store      *SuiStore
	toggleFunc func()
}

func NewHotkeyService(store *SuiStore, toggleFunc func()) *HotkeyService {
	hotkeyHandlerOnce.Do(func() {
		C.setupHandler()
	})

	svc := &HotkeyService{
		registered: make(map[string]C.EventHotKeyRef),
		store:      store,
		toggleFunc: toggleFunc,
	}

	// CGO 限制：C 侧无法直接持有 Go 的闭包/对象，这里用全局变量桥接。
	setGlobalHotkeyCallback(func(id uint32) {
		if id != hotkeyIDToggle {
			return
		}
		if svc.toggleFunc == nil {
			return
		}
		// 避免阻塞系统事件线程；如需强制主线程执行，请在 toggleFunc 内自行调度。
		go svc.toggleFunc()
	})

	return svc
}

func (hs *HotkeyService) RegisterToggleHotkey(keycode uint32, modifiers uint32) error {
	hotkeyHandlerOnce.Do(func() {
		C.setupHandler()
	})

	hs.mu.Lock()
	defer hs.mu.Unlock()

	// 先注销已存在的同名快捷键（失败也不阻断后续注册）。
	if existing, ok := hs.registered[toggleWindowTarget]; ok && existing != nil {
		if status := C.unregisterHotkey(existing); status != C.noErr {
			log.Printf("WARN 注销旧快捷键失败: target=%s status=%d", toggleWindowTarget, int32(status))
		}
		delete(hs.registered, toggleWindowTarget)
	}

	carbonModifiers := frontendToCarbonModifiers(modifiers)
	var ref C.EventHotKeyRef
	status := C.registerHotkey(C.UInt32(keycode), carbonModifiers, C.UInt32(hotkeyIDToggle), &ref)
	if status != C.noErr {
		err := fmt.Errorf("注册全局快捷键失败: status=%d keycode=%d modifiers=%d", int32(status), keycode, modifiers)
		log.Printf("WARN %v", err)
		return err
	}

	hs.registered[toggleWindowTarget] = ref
	return nil
}

func (hs *HotkeyService) UnregisterAll() {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	for key, ref := range hs.registered {
		if ref == nil {
			delete(hs.registered, key)
			continue
		}
		if status := C.unregisterHotkey(ref); status != C.noErr {
			log.Printf("WARN 注销快捷键失败: key=%s status=%d", key, int32(status))
		}
		delete(hs.registered, key)
	}
}

func (hs *HotkeyService) SaveToggleHotkey(keycode uint32, modifiers uint32) error {
	if hs.store == nil || hs.store.db == nil {
		err := errors.New("数据库未初始化")
		log.Printf("WARN 保存快捷键失败: %v", err)
		return err
	}

	tx, err := hs.store.db.Begin()
	if err != nil {
		log.Printf("WARN 保存快捷键失败: %v", err)
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var id int
	row := tx.QueryRow(`SELECT id FROM hotkeys WHERE target = ? LIMIT 1`, toggleWindowTarget)
	scanErr := row.Scan(&id)
	if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
		log.Printf("WARN 保存快捷键失败: %v", scanErr)
		return scanErr
	}

	description := "切换窗口"
	if scanErr == nil && id > 0 {
		_, err = tx.Exec(
			`UPDATE hotkeys SET keycode = ?, modifiers = ?, description = ? WHERE id = ?`,
			keycode,
			modifiers,
			description,
			id,
		)
	} else {
		_, err = tx.Exec(
			`INSERT INTO hotkeys (keycode, modifiers, description, target) VALUES (?, ?, ?, ?)`,
			keycode,
			modifiers,
			description,
			toggleWindowTarget,
		)
	}
	if err != nil {
		log.Printf("WARN 保存快捷键失败: %v", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Printf("WARN 保存快捷键失败: %v", err)
		return err
	}
	return nil
}

func (hs *HotkeyService) GetToggleHotkey() (*Hotkey, error) {
	if hs.store == nil || hs.store.db == nil {
		err := errors.New("数据库未初始化")
		log.Printf("WARN 读取快捷键失败: %v", err)
		return nil, err
	}

	var (
		id        int
		keycode   int64
		modifiers int64
	)
	row := hs.store.db.QueryRow(`SELECT id, keycode, modifiers FROM hotkeys WHERE target = ? LIMIT 1`, toggleWindowTarget)
	if err := row.Scan(&id, &keycode, &modifiers); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		log.Printf("WARN 读取快捷键失败: %v", err)
		return nil, err
	}
	return &Hotkey{
		ID:        id,
		KeyCode:   uint32(keycode),
		Modifiers: uint32(modifiers),
	}, nil
}

func (hs *HotkeyService) LoadToggleHotkey() error {
	hotkey, err := hs.GetToggleHotkey()
	if err != nil {
		// 静默失败：记录日志并返回 error，交由上层决定是否忽略。
		log.Printf("WARN 启动时加载快捷键失败: %v", err)
		return err
	}
	if hotkey == nil {
		return nil
	}
	if err := hs.RegisterToggleHotkey(hotkey.KeyCode, hotkey.Modifiers); err != nil {
		log.Printf("WARN 启动时注册快捷键失败: %v", err)
		return err
	}
	return nil
}

func frontendToCarbonModifiers(modifiers uint32) C.UInt32 {
	var carbon C.UInt32
	if modifiers&frontendModifierAlt != 0 {
		carbon |= C.UInt32(C.optionKey)
	}
	if modifiers&frontendModifierShift != 0 {
		carbon |= C.UInt32(C.shiftKey)
	}
	if modifiers&frontendModifierCmd != 0 {
		carbon |= C.UInt32(C.cmdKey)
	}
	if modifiers&frontendModifierCtrl != 0 {
		carbon |= C.UInt32(C.controlKey)
	}
	return carbon
}
