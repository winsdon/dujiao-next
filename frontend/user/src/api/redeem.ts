import { api } from './client'

export type RedeemVerifyResult = {
  valid: boolean
  product_id?: number
  product_name?: string
  plan_type?: string
  service_code?: string
  payment_region?: string
  status: string
  cooldown_remaining_sec: number
  masked_code?: string
  message?: string
  redeem_order_id?: number
}

export type RedeemSubmitResult = {
  redeem_order_id: number
  status: string
  message?: string
  cooldown_remaining_sec?: number
}

export type RedeemCaptcha = {
  id?: string
  url?: string | null
  expires_at?: string
}

export type RedeemStatusResult = {
  redeem_order_id?: number
  status: string
  stage?: string
  product_name?: string
  plan_type?: string
  service_code?: string
  payment_region?: string
  masked_code?: string
  masked_contact?: string
  masked_email?: string
  cooldown_remaining_sec?: number
  message?: string
  error_code?: string
  subscription_cancelled?: boolean | null
  completed_at?: string
  captcha?: RedeemCaptcha | null
  submitted_at?: string
}

export const redeemAPI = {
  verify: (code: string) =>
    api.post('/public/redeem/verify', { code }, { silentBusinessError: true }),
  submit: (payload: { code: string; contact: string; credential: string; authorized: boolean }) =>
    api.post('/public/redeem/submit', payload, { silentBusinessError: true }),
  status: (code: string) =>
    api.post('/public/redeem/status', { code }, { silentBusinessError: true }),
}
