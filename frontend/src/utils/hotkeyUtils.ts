// hotkeyUtils.ts

const modifierMap: Record<string, number> = {
  Control: 256,
  Alt: 512,
  Shift: 1024,
  Meta: 2048,
};

// 修饰键位定义（macOS 通常使用）
export const MODIFIERS = {
  CMD: 1 << 8,     // 256
  SHIFT: 1 << 9,   // 512
  ALT: 1 << 10,    // 1024
  CTRL: 1 << 11,   // 2048
};

// 快捷键字符串 => 键码映射（mac keycode 映射）
const keyMap: Record<string, number> = {
  // Letters
  A: 0, B: 11, C: 8, D: 2, E: 14, F: 3, G: 5,
  H: 4, I: 34, J: 38, K: 40, L: 37, M: 46,
  N: 45, O: 31, P: 35, Q: 12, R: 15, S: 1,
  T: 17, U: 32, V: 9, W: 13, X: 7, Y: 16, Z: 6,
  // Digits
  '0': 29, '1': 18, '2': 19, '3': 20, '4': 21, '5': 23, '6': 22, '7': 26, '8': 28, '9': 25,
  // Function keys
  F1: 122, F2: 120, F3: 99, F4: 118, F5: 96, F6: 97, F7: 98, F8: 100, F9: 101, F10: 109, F11: 103, F12: 111,
  // Others
  SPACE: 49, ENTER: 36, BACKSPACE: 51, TAB: 48, ESC: 53,
  UP: 126, DOWN: 125, LEFT: 123, RIGHT: 124,
};


export function parseHotkeyString(hotkey: string): { key: number, modifier: number } | null {
  if (!hotkey.includes('+')) return null;

  const parts = hotkey.split('+');
  const mods = parts.slice(0, -1); // 修饰键
  const keyStr = parts[parts.length - 1];

  let modifier = 0;
  for (const mod of mods) {
    if (modifierMap[mod]) modifier |= modifierMap[mod];
  }

  const key = keyMap[keyStr.toUpperCase()];
  if (key === undefined) return null;

  return { key, modifier };
}

export function parseShortcutToHotkey(shortcut: string): { key: number, modifier: number } {
  const parts = shortcut.toUpperCase().split('+');
  let keys = '';
  let modifier = 0;

  for (const part of parts) {
    switch (part) {
      case 'CMD':
      case 'COMMAND':
      case 'META':
        modifier |= 1 << 8;
        break;
      case 'SHIFT':
        modifier |= 1 << 9;
        break;
      case 'ALT':
      case 'OPTION':
        modifier |= 1 << 10;
        break;
      case 'CTRL':
      case 'CONTROL':
        modifier |= 1 << 11;
        break;
      default:
        keys = part;
    }
  }

  const key = keyMap[keys];
  return {
    key,
    modifier,
  };
}

// 反向映射：keycode => 字符
const reverseKeyMap = Object.fromEntries(
  Object.entries(keyMap).map(([k, v]) => [v, k])
);

// 将 keycode 和 modifier 转换为字符串（如 40, 768 => "Cmd+Shift+K"）
export function formatHotkeyString(key: number, modifier: number): string {
  const parts: string[] = [];

  if (modifier & MODIFIERS.CMD) parts.push('Cmd');
  if (modifier & MODIFIERS.SHIFT) parts.push('Shift');
  if (modifier & MODIFIERS.ALT) parts.push('Alt');
  if (modifier & MODIFIERS.CTRL) parts.push('Ctrl');

  const keyStr = reverseKeyMap[key];
  if (!keyStr) throw new Error(`Unknown keycode: ${key}`);

  parts.push(keyStr);
  return parts.join('+');
}

export function formatHotkeyStringmac(keycode: number, modifiers: number): string {
  const mods: string[] = [];

  if (modifiers & (1 << 8)) mods.push('Cmd');      // mac
  if (modifiers & (1 << 9)) mods.push('Shift');
  if (modifiers & (1 << 10)) mods.push('Alt');
  if (modifiers & (1 << 11)) mods.push('Control');

  const key = reverseKeyMap[keycode] ?? 'Unknown';

  return [...mods, key].join('+');
}