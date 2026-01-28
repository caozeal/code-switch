import * as ClaudeSettings from '../../bindings/codeswitch/services/claudesettingsservice'
import * as CodexSettings from '../../bindings/codeswitch/services/codexsettingsservice'
import * as GeminiSettings from '../../bindings/codeswitch/services/geminisettingsservice'
import * as OpenAISettings from '../../bindings/codeswitch/services/openaisettingsservice'
import type { ClaudeProxyStatus } from '../../bindings/codeswitch/services/models'

type Platform = 'claude' | 'codex' | 'gemini' | 'openai'

const services: Record<Platform, any> = {
  claude: ClaudeSettings,
  codex: CodexSettings,
  gemini: GeminiSettings,
  openai: OpenAISettings,
}

export const fetchProxyStatus = async (platform: Platform): Promise<ClaudeProxyStatus> => {
  return services[platform].ProxyStatus()
}

export const enableProxy = async (platform: Platform): Promise<void> => {
  await services[platform].EnableProxy()
}

export const disableProxy = async (platform: Platform): Promise<void> => {
  await services[platform].DisableProxy()
}
