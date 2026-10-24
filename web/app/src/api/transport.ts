// 聊天传输面契约：direct（Caddy mTLS 直连）为默认且唯一战略方向；relay
// 已弃用，仅作回滚杆保留（构建期注入 VITE_CHAT_TRANSPORT=relay 显式回退，
// 回滚杆：部署上一版镜像即回到中继）。relay 退役时收敛本联合，编译器将
// 指出全部待删分支。本模块零依赖：dashboard 首屏与重量级 chat 模块均可
// 安全引用。
export type ChatTransport = 'direct' | 'relay'

export function chatTransport(): ChatTransport {
  return import.meta.env.VITE_CHAT_TRANSPORT === 'relay' ? 'relay' : 'direct'
}
