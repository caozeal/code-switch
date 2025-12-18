import { Call } from '@wailsio/runtime'

export type HotkeyConfig = { id: number; keycode: number; modifiers: number }

const HOTKEY_SERVICE = 'codeswitch/services.HotkeyService'

export const registerToggleHotkey = async (keycode: number, modifiers: number): Promise<void> => {
  await Call.ByName(`${HOTKEY_SERVICE}.RegisterToggleHotkey`, keycode, modifiers)
}

export const saveToggleHotkey = async (keycode: number, modifiers: number): Promise<void> => {
  await Call.ByName(`${HOTKEY_SERVICE}.SaveToggleHotkey`, keycode, modifiers)
  await registerToggleHotkey(keycode, modifiers)
}

export const getToggleHotkey = async (): Promise<HotkeyConfig | null> => {
  const data = await Call.ByName(`${HOTKEY_SERVICE}.GetToggleHotkey`)
  return (data as HotkeyConfig | null) ?? null
}
