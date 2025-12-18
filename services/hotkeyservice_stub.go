//go:build !darwin

package services

import (
	"database/sql"
	"errors"
)

const toggleWindowTarget = "toggle_window"

type HotkeyService struct {
	store      *SuiStore
	toggleFunc func()
}

func NewHotkeyService(store *SuiStore, toggleFunc func()) *HotkeyService {
	return &HotkeyService{
		store:      store,
		toggleFunc: toggleFunc,
	}
}

func (hs *HotkeyService) RegisterToggleHotkey(keycode uint32, modifiers uint32) error {
	return errors.New("当前系统暂不支持全局快捷键")
}

func (hs *HotkeyService) UnregisterAll() {}

func (hs *HotkeyService) SaveToggleHotkey(keycode uint32, modifiers uint32) error {
	if hs.store == nil || hs.store.db == nil {
		return errors.New("数据库未初始化")
	}

	tx, err := hs.store.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var id int
	row := tx.QueryRow(`SELECT id FROM hotkeys WHERE target = ? LIMIT 1`, toggleWindowTarget)
	scanErr := row.Scan(&id)
	if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
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
		return err
	}

	return tx.Commit()
}

func (hs *HotkeyService) GetToggleHotkey() (*Hotkey, error) {
	if hs.store == nil || hs.store.db == nil {
		return nil, errors.New("数据库未初始化")
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
		return nil, err
	}
	return &Hotkey{
		ID:        id,
		KeyCode:   uint32(keycode),
		Modifiers: uint32(modifiers),
	}, nil
}

func (hs *HotkeyService) LoadToggleHotkey() error {
	return nil
}
