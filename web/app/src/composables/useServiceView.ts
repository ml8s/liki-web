import { onMounted } from 'vue'
import { useProfilesStore } from '@/stores/profiles'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useChatStore } from '@/stores/chat'

/**
 * 服务视图公共逻辑：加载档案、权益、会话列表。
 * 各视图 onMounted 时调用即可，无需重复 await Promise.all。
 */
export function useServiceView() {
  const profiles = useProfilesStore()
  const entitlements = useEntitlementsStore()
  const chat = useChatStore()

  onMounted(() => {
    // 非阻塞并行加载，不 await——避免阻塞视图首屏渲染。
    profiles.loadList()
    entitlements.load()
    chat.loadSessions()
  })

  return { profiles, entitlements, chat }
}
