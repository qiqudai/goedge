import request from '@/utils/request'
import type { GlobalConfigPayload } from '@/types/errorPages'

export function fetchGlobalConfig() {
  return request.get('/global_config')
}

export function saveGlobalConfig(payload: GlobalConfigPayload) {
  return request.post('/global_config', payload)
}
