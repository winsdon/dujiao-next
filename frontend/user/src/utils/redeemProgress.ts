// 兑换「开通中」步骤的纯展示逻辑：阶段判定、时间线节点状态、等待时长格式化。

export type RedeemPhase = 'processing' | 'captcha' | 'confirming'
export type RedeemNodeState = 'done' | 'active' | 'attention' | 'paused' | 'idle'

export type RedeemTimelineStates = {
  submitted: RedeemNodeState
  processing: RedeemNodeState
  captcha: RedeemNodeState
  confirm: RedeemNodeState
}

/** 正常开通的典型耗时上限（秒），超过后提示「比平时稍慢」。 */
export const REDEEM_SLOW_THRESHOLD_SEC = 180

export function resolveRedeemPhase(status: string | undefined, hasCaptcha: boolean): RedeemPhase {
  if (hasCaptcha) return 'captcha'
  if (status === 'pending_confirm') return 'confirming'
  return 'processing'
}

export function resolveTimelineStates(phase: RedeemPhase): RedeemTimelineStates {
  switch (phase) {
    case 'captcha':
      return { submitted: 'done', processing: 'paused', captcha: 'attention', confirm: 'idle' }
    case 'confirming':
      return { submitted: 'done', processing: 'done', captcha: 'idle', confirm: 'active' }
    default:
      return { submitted: 'done', processing: 'active', captcha: 'idle', confirm: 'idle' }
  }
}

/** 解析后端返回的 `2006-01-02 15:04:05`（固定 UTC+8）时间，返回毫秒时间戳。 */
export function parseCstDateTime(value?: string | null): number | null {
  const raw = String(value || '').trim()
  const match = raw.match(/^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2}:\d{2})$/)
  if (!match) return null
  const ms = Date.parse(`${match[1]}T${match[2]}+08:00`)
  return Number.isNaN(ms) ? null : ms
}

/** 秒数格式化为 mm:ss，超过一小时为 h:mm:ss；负数按 0 处理。 */
export function formatElapsed(totalSec: number): string {
  const sec = Math.max(0, Math.floor(totalSec))
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  const pad = (n: number) => String(n).padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`
}
