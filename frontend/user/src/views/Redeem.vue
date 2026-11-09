<template>
  <div class="min-h-screen bg-background pt-24 pb-16 text-foreground">
    <div class="mx-auto w-full max-w-[520px] px-4">
      <RedeemStepper :labels="stepLabels" :current="stepIndex" />

      <div class="overflow-hidden rounded-2xl border bg-card p-6 shadow-sm sm:p-8">
        <Transition
          mode="out-in"
          enter-active-class="transition duration-300 ease-out motion-reduce:transition-none"
          enter-from-class="translate-y-2 opacity-0"
          leave-active-class="transition duration-150 ease-in motion-reduce:transition-none"
          leave-to-class="opacity-0"
        >
          <!-- Query mode -->
          <div v-if="mode === 'query'" key="query">
            <div class="flex items-start gap-3">
              <span class="flex size-11 shrink-0 items-center justify-center rounded-xl bg-primary-soft text-primary">
                <Search class="size-5" />
              </span>
              <div>
                <h1 class="text-xl font-bold">{{ t('redeem.queryTitle') }}</h1>
                <p class="mt-1 text-sm text-muted-foreground">{{ t('redeem.queryHint') }}</p>
              </div>
            </div>
            <div class="relative mt-6">
              <KeyRound class="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                v-model="code"
                class="h-12 pl-10 font-mono"
                :placeholder="t('redeem.codePlaceholder')"
                @keyup.enter="handleQuery"
              />
            </div>
            <Alert v-if="error" variant="destructive" class="mt-4">
              <AlertDescription>{{ error }}</AlertDescription>
            </Alert>
            <div class="mt-5 flex flex-col gap-2 sm:flex-row-reverse">
              <Button class="h-11 font-semibold sm:flex-1" :disabled="busy || !code.trim()" @click="handleQuery">
                <Loader2 v-if="busy" class="motion-safe:animate-spin" />
                {{ busy ? t('redeem.loading') : t('redeem.queryBtn') }}
              </Button>
              <Button variant="outline" class="h-11 sm:flex-1" @click="backToVerify">
                <ArrowLeft />
                {{ t('redeem.backRedeem') }}
              </Button>
            </div>
          </div>

          <!-- Step 1: verify -->
          <div v-else-if="step === 'verify'" key="verify">
            <div class="flex items-start gap-3">
              <span class="flex size-11 shrink-0 items-center justify-center rounded-xl bg-primary-soft text-primary">
                <Ticket class="size-5" />
              </span>
              <div>
                <h1 class="text-xl font-bold">{{ t('redeem.title') }}</h1>
                <p class="mt-1 text-sm text-muted-foreground">{{ t('redeem.subtitle') }}</p>
              </div>
            </div>
            <div class="relative mt-6">
              <KeyRound class="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                v-model="code"
                class="h-12 pl-10 font-mono"
                :placeholder="t('redeem.codePlaceholder')"
                @keyup.enter="handleVerify"
              />
            </div>
            <p
              v-if="verifyInfo?.valid"
              class="mt-2 inline-flex items-center gap-1 rounded-full bg-success-soft px-2.5 py-1 text-xs text-success"
            >
              <Check class="size-3.5" stroke-width="3" />
              {{ t('redeem.valid') }} · {{ verifyInfo?.product_name }} · {{ statusLabel(verifyInfo?.status) }}
            </p>
            <Alert v-if="error" variant="destructive" class="mt-4">
              <AlertDescription>{{ error }}</AlertDescription>
            </Alert>
            <div
              v-if="cooldownSec > 0"
              class="mt-4 flex items-center gap-2 rounded-xl border border-warning/40 bg-warning-soft px-3 py-2.5 text-sm"
            >
              <Clock class="size-4 shrink-0 text-warning" />
              <span class="tabular-nums">{{ t('redeem.cooldown', { sec: cooldownSec }) }}</span>
            </div>
            <Button
              class="mt-5 h-11 w-full font-semibold"
              :disabled="busy || !code.trim() || cooldownSec > 0"
              @click="handleVerify"
            >
              <Loader2 v-if="busy" class="motion-safe:animate-spin" />
              {{ busy ? t('redeem.loading') : t('redeem.next') }}
              <ArrowRight v-if="!busy" />
            </Button>
            <div class="mt-6 border-t pt-4 text-center">
              <button
                type="button"
                class="text-sm text-muted-foreground transition-colors hover:text-primary"
                @click="openQuery"
              >
                {{ t('redeem.queryLink') }}
              </button>
            </div>
          </div>

          <!-- Step 2: credentials -->
          <div v-else-if="step === 'cred'" key="cred">
            <h1 class="text-xl font-bold">{{ t('redeem.credTitle') }}</h1>
            <div class="mt-4 flex items-center justify-between gap-3 rounded-xl border bg-muted/40 px-4 py-3">
              <div class="min-w-0">
                <div class="truncate font-medium">{{ verifyInfo?.product_name }}</div>
                <div class="mt-0.5 font-mono text-xs text-muted-foreground">{{ verifyInfo?.masked_code }}</div>
              </div>
              <Badge v-if="verifyInfo?.plan_type" variant="accent" class="shrink-0">{{ verifyInfo?.plan_type }}</Badge>
            </div>

            <label for="redeem-contact" class="mt-5 block text-sm font-medium">{{ t('redeem.contactLabel') }}</label>
            <div class="relative mt-1.5">
              <Mail class="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                id="redeem-contact"
                v-model="contact"
                class="h-11 pl-10"
                :placeholder="t('redeem.contactPlaceholder')"
              />
            </div>

            <div class="mt-5 rounded-xl border bg-muted/30 p-4 text-xs leading-relaxed text-muted-foreground">
              <template v-if="isGrok">
                {{ t('redeem.grokHint') }}
              </template>
              <ol v-else class="space-y-2">
                <li v-for="(stepText, i) in gptSteps" :key="i" class="flex gap-2.5">
                  <span
                    class="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary-soft text-[11px] font-semibold text-primary"
                  >
                    {{ i + 1 }}
                  </span>
                  <span class="pt-0.5">
                    {{ stepText }}
                    <a
                      v-if="i === 1"
                      href="https://chatgpt.com/api/auth/session"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="mt-1 inline-flex items-center gap-1 font-mono font-semibold text-primary hover:underline"
                    >
                      chatgpt.com/api/auth/session
                      <ExternalLink class="size-3" />
                    </a>
                  </span>
                </li>
              </ol>
            </div>

            <label for="redeem-credential" class="mt-5 block text-sm font-medium">
              {{ isGrok ? t('redeem.ssoLabel') : t('redeem.sessionLabel') }}
            </label>
            <Textarea
              id="redeem-credential"
              v-model="credential"
              class="mt-1.5 min-h-[120px] font-mono text-xs transition-colors"
              :class="credentialBorderClass"
              :placeholder="isGrok ? t('redeem.ssoPlaceholder') : t('redeem.sessionPlaceholder')"
            />
            <p v-if="credHint" class="mt-2 text-xs text-success">{{ credHint }}</p>
            <p v-if="credError" class="mt-2 text-xs text-destructive">{{ credError }}</p>

            <label class="mt-5 flex cursor-pointer items-start gap-2.5 text-sm">
              <input v-model="authorized" type="checkbox" class="mt-0.5 size-4 shrink-0 accent-primary" />
              <span>{{ t('redeem.authorize') }}</span>
            </label>
            <p class="mt-2 flex gap-1.5 text-xs leading-relaxed text-muted-foreground">
              <Lock class="mt-0.5 size-3.5 shrink-0" />
              <span>{{ t('redeem.privacy') }}</span>
            </p>

            <Alert v-if="error" variant="destructive" class="mt-4">
              <AlertDescription>{{ error }}</AlertDescription>
            </Alert>

            <div class="mt-6 flex flex-col gap-2 sm:flex-row-reverse">
              <Button class="h-11 font-semibold sm:flex-1" :disabled="busy || !canSubmit" @click="handleSubmit">
                <Loader2 v-if="busy" class="motion-safe:animate-spin" />
                {{ busy ? t('redeem.submitting') : t('redeem.submit') }}
              </Button>
              <Button variant="outline" class="h-11 sm:flex-1" :disabled="busy" @click="step = 'verify'">
                <ArrowLeft />
                {{ t('redeem.prev') }}
              </Button>
            </div>
          </div>

          <!-- Step 3: processing -->
          <div v-else-if="step === 'proc'" key="proc">
            <RedeemProgress
              :status="status"
              :product-name="status?.product_name || verifyInfo?.product_name"
              :masked-code="status?.masked_code || verifyInfo?.masked_code"
              :next-refresh-in="nextRefreshIn"
              :refreshing="refreshing"
              :fallback-started-at="procStartedAt"
              @refresh="refreshNow"
              @open-captcha="openCaptcha"
            />
            <Alert v-if="error" variant="destructive" class="mt-4">
              <AlertDescription>{{ error }}</AlertDescription>
            </Alert>
          </div>

          <!-- Step 4: result -->
          <div v-else key="result">
            <RedeemResult
              :kind="resultKind"
              :status="status"
              :verify-info="verifyInfo"
              :cooldown-sec="cooldownSec"
              :error="error"
              @retry="retryCred"
            />
          </div>
        </Transition>
      </div>

      <p class="mt-6 flex items-center justify-center gap-1.5 text-center text-xs text-muted-foreground">
        <LifeBuoy class="size-3.5" />
        {{ t('redeem.csHint') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Clock,
  ExternalLink,
  KeyRound,
  LifeBuoy,
  Loader2,
  Lock,
  Mail,
  Search,
  Ticket,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  redeemAPI,
  type RedeemStatusResult,
  type RedeemVerifyResult,
} from '@/api/redeem'
import RedeemStepper from './redeem/RedeemStepper.vue'
import RedeemProgress from './redeem/RedeemProgress.vue'
import RedeemResult from './redeem/RedeemResult.vue'

const POLL_INTERVAL_SEC = 5

const { t } = useI18n()
const route = useRoute()

const mode = ref<'redeem' | 'query'>('redeem')
const step = ref<'verify' | 'cred' | 'proc' | 'result'>('verify')
const code = ref('')
const contact = ref('')
const credential = ref('')
const authorized = ref(false)
const busy = ref(false)
const error = ref('')
const verifyInfo = ref<RedeemVerifyResult | null>(null)
const status = ref<RedeemStatusResult | null>(null)
const cooldownSec = ref(0)
const credHint = ref('')
const credError = ref('')
const nextRefreshIn = ref(POLL_INTERVAL_SEC)
const refreshing = ref(false)
const procStartedAt = ref(Date.now())

let cooldownTimer: number | undefined
let pollTimer: number | undefined

const stepLabels = computed(() => [
  t('redeem.step1'),
  t('redeem.step2'),
  t('redeem.step3'),
  t('redeem.step4'),
])

const gptSteps = computed(() => [t('redeem.gptStep1'), t('redeem.gptStep2'), t('redeem.gptStep3')])

const stepIndex = computed(() => {
  if (step.value === 'verify') return 0
  if (step.value === 'cred') return 1
  if (step.value === 'proc') return 2
  // 开通成功时四步全部完成
  return resultKind.value === 'ok' ? 4 : 3
})

const isGrok = computed(() => (verifyInfo.value?.service_code || '').toLowerCase() === 'grok')

const canSubmit = computed(() => {
  return (
    authorized.value &&
    contact.value.trim().length > 0 &&
    credential.value.trim().length > 0 &&
    !credError.value
  )
})

const credentialBorderClass = computed(() => {
  if (credError.value) return 'border-destructive focus-visible:ring-destructive/40'
  if (credHint.value) return 'border-success focus-visible:ring-success/40'
  return ''
})

const resultKind = computed<'ok' | 'fail' | 'pending'>(() => {
  const s = status.value?.status
  if (s === 'succeeded') return 'ok'
  if (s === 'pending_confirm') return 'pending'
  return 'fail'
})

function statusLabel(s?: string) {
  if (!s) return ''
  const map: Record<string, string> = {
    unused: t('redeem.statusUnused'),
    cooldown: t('redeem.statusCooldown'),
    processing: t('redeem.statusProcessing'),
    pending_confirm: t('redeem.statusPending'),
    succeeded: t('redeem.statusSucceeded'),
    failed: t('redeem.statusFailed'),
    locked: t('redeem.statusLocked'),
  }
  return map[s] || s
}

function normalizeCode(v: string) {
  return v.replace(/\s+/g, '').trim()
}

function startCooldown(sec: number) {
  cooldownSec.value = Math.max(0, sec)
  if (cooldownTimer) window.clearInterval(cooldownTimer)
  if (cooldownSec.value <= 0) return
  cooldownTimer = window.setInterval(() => {
    cooldownSec.value -= 1
    if (cooldownSec.value <= 0 && cooldownTimer) {
      window.clearInterval(cooldownTimer)
      cooldownTimer = undefined
    }
  }, 1000)
}

function validateContact(v: string): boolean {
  const s = v.trim()
  if (!s) return false
  if (s.includes('@')) return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s)
  // Telegram: @user or t.me/user or plain handle
  return s.length >= 3
}

function validateCredential() {
  credHint.value = ''
  credError.value = ''
  const raw = credential.value.trim()
  if (!raw) return
  if (raw.length > 64 * 1024) {
    credError.value = t('redeem.credTooLarge')
    return
  }
  if (isGrok.value) {
    // Accept raw SSO string or JSON with sso / cookieHeader
    try {
      if (raw.startsWith('{')) {
        const j = JSON.parse(raw)
        if (!(j.sso || j['sso-rw'] || j.cookieHeader || j.cookie)) {
          credError.value = t('redeem.grokInvalid')
          return
        }
      }
      credHint.value = t('redeem.grokOk')
    } catch {
      // raw string ok for grok
      if (raw.length < 8) {
        credError.value = t('redeem.grokInvalid')
        return
      }
      credHint.value = t('redeem.grokOk')
    }
    return
  }
  try {
    const j = JSON.parse(raw)
    if (!j.accessToken || !j.user) {
      credError.value = t('redeem.sessionInvalid')
      return
    }
    const email = j.user?.email || j.user?.email_address || ''
    const masked = email ? email[0] + '***@' + String(email).split('@')[1] : ''
    credHint.value = masked
      ? t('redeem.sessionOk', { email: masked })
      : t('redeem.sessionOkNoEmail')
  } catch {
    credError.value = t('redeem.sessionInvalid')
  }
}

watch(credential, () => validateCredential())
watch(
  () => verifyInfo.value?.service_code,
  () => validateCredential(),
)
watch(step, (v, prev) => {
  if (v === 'proc' && prev !== 'proc') procStartedAt.value = Date.now()
})

function applyStatus(res: RedeemStatusResult) {
  status.value = res
  if (res.cooldown_remaining_sec && res.cooldown_remaining_sec > 0) {
    startCooldown(res.cooldown_remaining_sec)
  }
  if (res.status === 'succeeded' || res.status === 'failed' || res.status === 'pending_confirm') {
    stopPoll()
    step.value = 'result'
    mode.value = 'redeem'
  } else if (['processing', 'locked', 'pending_confirm'].includes(res.status) || res.captcha?.url) {
    step.value = 'proc'
    mode.value = 'redeem'
  } else if (res.status === 'cooldown') {
    step.value = 'result'
    // treat as fail with cooldown for retry
  }
}

async function handleVerify() {
  error.value = ''
  busy.value = true
  code.value = normalizeCode(code.value)
  try {
    const { data } = await redeemAPI.verify(code.value)
    const res = (data?.data || data) as RedeemVerifyResult
    verifyInfo.value = res
    if (!res.valid) {
      error.value = res.message || t('redeem.invalidCode')
      if (res.cooldown_remaining_sec) startCooldown(res.cooldown_remaining_sec)
      return
    }
    if (res.status === 'succeeded') {
      // jump to status result
      await fetchStatus(true)
      return
    }
    if (res.status === 'processing' || res.status === 'locked' || res.status === 'pending_confirm') {
      await fetchStatus(true)
      startPoll()
      return
    }
    if (res.status === 'cooldown' && res.cooldown_remaining_sec > 0) {
      startCooldown(res.cooldown_remaining_sec)
      error.value = res.message || t('redeem.cooldown', { sec: res.cooldown_remaining_sec })
      return
    }
    if (res.status === 'unmapped') {
      error.value = res.message || t('redeem.unmapped')
      return
    }
    step.value = 'cred'
  } catch (e: any) {
    error.value = e?.message || t('redeem.networkError')
  } finally {
    busy.value = false
  }
}

async function handleSubmit() {
  error.value = ''
  if (!validateContact(contact.value)) {
    error.value = t('redeem.contactInvalid')
    return
  }
  validateCredential()
  if (credError.value || !credential.value.trim()) {
    error.value = credError.value || t('redeem.credRequired')
    return
  }
  if (!authorized.value) {
    error.value = t('redeem.authorizeRequired')
    return
  }
  busy.value = true
  try {
    const { data } = await redeemAPI.submit({
      code: normalizeCode(code.value),
      contact: contact.value.trim(),
      credential: credential.value.trim(),
      authorized: true,
    })
    const res = data?.data || data
    if (res?.cooldown_remaining_sec) startCooldown(res.cooldown_remaining_sec)
    // Clear credential from memory ASAP
    credential.value = ''
    step.value = 'proc'
    await fetchStatus(true)
    startPoll()
  } catch (e: any) {
    // Keep the credential in the form (memory only) so the user can retry after maintenance.
    error.value = e?.message || t('redeem.networkError')
  } finally {
    busy.value = false
  }
}

async function fetchStatus(forceResult = false) {
  try {
    const { data } = await redeemAPI.status(normalizeCode(code.value))
    const res = (data?.data || data) as RedeemStatusResult
    // 轮询成功后清掉之前的临时网络错误，避免过期报错一直挂在进度页上
    error.value = ''
    applyStatus(res)
    if (forceResult && (res.status === 'succeeded' || res.status === 'failed')) {
      step.value = 'result'
    }
  } catch (e: any) {
    error.value = e?.message || t('redeem.networkError')
  }
}

async function handleQuery() {
  error.value = ''
  busy.value = true
  code.value = normalizeCode(code.value)
  try {
    await fetchStatus(true)
    if (status.value && ['processing', 'locked'].includes(status.value.status)) {
      startPoll()
    }
  } finally {
    busy.value = false
  }
}

async function refreshNow() {
  if (refreshing.value) return
  refreshing.value = true
  try {
    await fetchStatus()
  } finally {
    refreshing.value = false
    nextRefreshIn.value = POLL_INTERVAL_SEC
  }
}

function startPoll() {
  stopPoll()
  nextRefreshIn.value = POLL_INTERVAL_SEC
  // 每秒走一次倒计时，归零时拉取状态；请求进行中不重复发起
  pollTimer = window.setInterval(() => {
    if (refreshing.value) return
    nextRefreshIn.value -= 1
    if (nextRefreshIn.value <= 0) void refreshNow()
  }, 1000)
}

function stopPoll() {
  if (pollTimer) {
    window.clearInterval(pollTimer)
    pollTimer = undefined
  }
}

function openCaptcha() {
  const url = status.value?.captcha?.url
  if (url) window.open(url, '_blank', 'noopener,noreferrer')
}

function retryCred() {
  error.value = ''
  credential.value = ''
  authorized.value = false
  step.value = 'cred'
  status.value = null
}

function openQuery() {
  mode.value = 'query'
  error.value = ''
  step.value = 'verify'
}

function backToVerify() {
  mode.value = 'redeem'
  step.value = 'verify'
  error.value = ''
  stopPoll()
}

onMounted(() => {
  if (route.query.q === '1' || route.path.endsWith('/query')) {
    mode.value = 'query'
  }
  const c = String(route.query.code || '').trim()
  if (c) code.value = c
})

onBeforeUnmount(() => {
  stopPoll()
  if (cooldownTimer) window.clearInterval(cooldownTimer)
  credential.value = ''
})
</script>
