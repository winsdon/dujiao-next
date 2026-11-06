<template>
  <div class="min-h-screen bg-background text-foreground pt-24 pb-16">
    <div class="mx-auto w-full max-w-[480px] px-4">
      <!-- Stepper -->
      <div class="mb-8 flex flex-wrap items-center gap-2 text-sm">
        <div
          v-for="(label, i) in stepLabels"
          :key="label"
          class="flex items-center gap-1.5"
          :class="i <= stepIndex ? 'text-foreground' : 'text-muted-foreground'"
        >
          <span
            class="inline-flex h-[22px] w-[22px] items-center justify-center rounded-full text-xs"
            :class="stepCircleClass(i)"
          >
            <span v-if="i < stepIndex">✓</span>
            <span v-else>{{ i + 1 }}</span>
          </span>
          <span>{{ label }}</span>
        </div>
      </div>

      <div class="rounded-2xl border bg-card p-6 shadow-sm transition-opacity duration-200">
        <!-- Query mode -->
        <template v-if="mode === 'query'">
          <h1 class="text-xl font-bold">{{ t('redeem.queryTitle') }}</h1>
          <p class="mt-2 text-sm text-muted-foreground">{{ t('redeem.queryHint') }}</p>
          <Input
            v-model="code"
            class="mt-4 h-11 font-mono"
            :placeholder="t('redeem.codePlaceholder')"
            @keyup.enter="handleQuery"
          />
          <Alert v-if="error" variant="destructive" class="mt-4">
            <AlertDescription>{{ error }}</AlertDescription>
          </Alert>
          <div class="mt-4 flex flex-col gap-2 sm:flex-row">
            <Button class="font-bold sm:flex-1" :style="accentStyle" :disabled="busy" @click="handleQuery">
              {{ busy ? t('redeem.loading') : t('redeem.queryBtn') }}
            </Button>
            <Button variant="outline" class="sm:flex-1" @click="backToVerify">{{ t('redeem.backRedeem') }}</Button>
          </div>
        </template>

        <!-- Step 1: verify -->
        <template v-else-if="step === 'verify'">
          <h1 class="text-xl font-bold">{{ t('redeem.title') }}</h1>
          <p class="mt-2 text-sm text-muted-foreground">{{ t('redeem.subtitle') }}</p>
          <Input
            v-model="code"
            class="mt-4 h-11 font-mono"
            :placeholder="t('redeem.codePlaceholder')"
            @keyup.enter="handleVerify"
          />
          <p v-if="verifyInfo?.valid" class="mt-2 text-xs text-emerald-600 dark:text-emerald-400">
            ✓ {{ t('redeem.valid') }} · {{ verifyInfo?.product_name }} · {{ statusLabel(verifyInfo?.status) }}
          </p>
          <Alert v-if="error" variant="destructive" class="mt-4">
            <AlertDescription>{{ error }}</AlertDescription>
          </Alert>
          <Alert v-if="cooldownSec > 0" class="mt-4">
            <AlertDescription>{{ t('redeem.cooldown', { sec: cooldownSec }) }}</AlertDescription>
          </Alert>
          <Button
            class="mt-4 w-full font-bold text-white"
            :style="accentStyle"
            :disabled="busy || !code.trim() || cooldownSec > 0"
            @click="handleVerify"
          >
            {{ busy ? t('redeem.loading') : t('redeem.next') }}
          </Button>
          <button
            type="button"
            class="mt-4 text-sm text-muted-foreground hover:text-foreground"
            @click="mode = 'query'; error = ''; step = 'verify'"
          >
            {{ t('redeem.queryLink') }}
          </button>
        </template>

        <!-- Step 2: credentials -->
        <template v-else-if="step === 'cred'">
          <h1 class="text-xl font-bold">{{ t('redeem.credTitle') }}</h1>
          <div class="mt-3 rounded-xl border bg-muted/40 px-3 py-2 text-sm">
            <div class="font-medium">{{ verifyInfo?.product_name }}</div>
            <div class="font-mono text-xs text-muted-foreground">{{ verifyInfo?.masked_code }} · {{ verifyInfo?.plan_type }}</div>
          </div>

          <label class="mt-4 block text-sm text-muted-foreground">{{ t('redeem.contactLabel') }}</label>
          <Input v-model="contact" class="mt-1 h-11" :placeholder="t('redeem.contactPlaceholder')" />

          <div class="mt-4 rounded-xl border bg-muted/30 p-3 text-xs leading-relaxed text-muted-foreground">
            <template v-if="isGrok">
              {{ t('redeem.grokHint') }}
            </template>
            <template v-else>
              {{ t('redeem.gptHint') }}
              <a
                href="https://chatgpt.com/api/auth/session"
                target="_blank"
                rel="noopener noreferrer"
                class="ml-1 font-semibold underline"
                :style="{ color: accent }"
              >chatgpt.com/api/auth/session ↗</a>
            </template>
          </div>

          <label class="mt-3 block text-sm text-muted-foreground">{{ isGrok ? t('redeem.ssoLabel') : t('redeem.sessionLabel') }}</label>
          <Textarea
            v-model="credential"
            class="mt-1 min-h-[120px] font-mono text-xs"
            :placeholder="isGrok ? t('redeem.ssoPlaceholder') : t('redeem.sessionPlaceholder')"
          />
          <p v-if="credHint" class="mt-2 text-xs text-emerald-600 dark:text-emerald-400">{{ credHint }}</p>
          <p v-if="credError" class="mt-2 text-xs text-destructive">{{ credError }}</p>

          <label class="mt-4 flex cursor-pointer items-start gap-2 text-sm">
            <input v-model="authorized" type="checkbox" class="mt-1 size-4 accent-[#6E56CF]" />
            <span>{{ t('redeem.authorize') }}</span>
          </label>
          <p class="mt-2 text-xs text-muted-foreground">{{ t('redeem.privacy') }}</p>

          <Alert v-if="error" variant="destructive" class="mt-4">
            <AlertDescription>{{ error }}</AlertDescription>
          </Alert>

          <div class="mt-4 flex flex-col gap-2 sm:flex-row">
            <Button variant="outline" class="sm:flex-1" :disabled="busy" @click="step = 'verify'">{{ t('redeem.prev') }}</Button>
            <Button
              class="font-bold text-white sm:flex-1"
              :style="accentStyle"
              :disabled="busy || !canSubmit"
              @click="handleSubmit"
            >
              {{ busy ? t('redeem.submitting') : t('redeem.submit') }}
            </Button>
          </div>
        </template>

        <!-- Step 3: processing -->
        <template v-else-if="step === 'proc'">
          <h1 class="text-xl font-bold">{{ t('redeem.procTitle') }}</h1>
          <p class="mt-2 text-sm text-muted-foreground">{{ t('redeem.procHint') }}</p>
          <div class="mt-4 space-y-0 text-sm">
            <div class="flex justify-between border-b py-2">
              <span class="text-muted-foreground">{{ t('redeem.stageSubmitted') }}</span>
              <span class="text-emerald-600">✓</span>
            </div>
            <div class="flex justify-between border-b py-2">
              <span class="text-muted-foreground">{{ t('redeem.stageUpstream') }}</span>
              <span :style="{ color: accent }">{{ stageUpstreamLabel }}</span>
            </div>
            <div class="flex justify-between border-b py-2">
              <span class="text-muted-foreground">{{ t('redeem.stageCaptcha') }}</span>
              <span :class="status?.captcha?.url ? 'text-amber-600' : 'text-muted-foreground'">
                {{ status?.captcha?.url ? t('redeem.captchaNeeded') : '—' }}
              </span>
            </div>
          </div>

          <div v-if="status?.captcha?.url" class="mt-4 rounded-xl border border-amber-500/40 bg-amber-500/5 p-4">
            <div class="font-semibold">{{ t('redeem.captchaTitle') }}</div>
            <p class="mt-1 text-xs text-muted-foreground">{{ t('redeem.captchaHint') }}</p>
            <Button class="mt-3 font-bold text-white" :style="accentStyle" @click="openCaptcha">
              {{ t('redeem.captchaBtn') }} ↗
            </Button>
          </div>

          <p class="mt-4 text-xs text-muted-foreground">
            {{ t('redeem.lastRefresh', { time: lastRefreshLabel }) }}
          </p>
          <Alert v-if="error" variant="destructive" class="mt-4">
            <AlertDescription>{{ error }}</AlertDescription>
          </Alert>
        </template>

        <!-- Step 4: result -->
        <template v-else-if="step === 'result'">
          <template v-if="resultKind === 'ok'">
            <h1 class="text-xl font-bold text-emerald-600 dark:text-emerald-400">✓ {{ t('redeem.okTitle') }}</h1>
            <div class="mt-4 space-y-0 text-sm">
              <div class="flex justify-between border-b py-2">
                <span class="text-muted-foreground">{{ t('redeem.plan') }}</span>
                <span class="font-mono">{{ status?.product_name || verifyInfo?.product_name }}</span>
              </div>
              <div class="flex justify-between border-b py-2">
                <span class="text-muted-foreground">{{ t('redeem.account') }}</span>
                <span class="font-mono">{{ status?.masked_email || t('redeem.emailPending') }}</span>
              </div>
              <div class="flex justify-between border-b py-2">
                <span class="text-muted-foreground">{{ t('redeem.region') }}</span>
                <span class="font-mono">{{ status?.payment_region || '—' }}</span>
              </div>
              <div class="flex justify-between border-b py-2">
                <span class="text-muted-foreground">{{ t('redeem.completedAt') }}</span>
                <span class="font-mono">{{ formatCompleted(status?.completed_at) }}</span>
              </div>
              <div class="flex justify-between border-b py-2">
                <span class="text-muted-foreground">{{ t('redeem.autoRenew') }}</span>
                <span :class="status?.subscription_cancelled === true ? 'text-emerald-600' : 'text-amber-600'">
                  {{ autoRenewLabel }}
                </span>
              </div>
              <div class="flex justify-between border-b py-2">
                <span class="text-muted-foreground">{{ t('redeem.code') }}</span>
                <span class="font-mono">{{ status?.masked_code || verifyInfo?.masked_code }} {{ t('redeem.used') }}</span>
              </div>
            </div>
            <div v-if="status?.subscription_cancelled !== true" class="mt-3 rounded-xl border bg-muted/40 p-3 text-xs text-muted-foreground">
              {{ t('redeem.autoRenewHint') }}
            </div>
          </template>

          <template v-else-if="resultKind === 'pending'">
            <h1 class="text-xl font-bold text-amber-600">{{ t('redeem.pendingTitle') }}</h1>
            <p class="mt-3 rounded-xl border bg-muted/40 p-3 text-sm text-muted-foreground">
              {{ status?.message || t('redeem.pendingHint') }}
            </p>
          </template>

          <template v-else>
            <h1 class="text-xl font-bold text-destructive">{{ t('redeem.failTitle') }}</h1>
            <p class="mt-3 rounded-xl border bg-muted/40 p-3 text-sm text-muted-foreground">
              {{ status?.message || error || t('redeem.failHint') }}
            </p>
            <p v-if="cooldownSec > 0" class="mt-2 text-xs text-muted-foreground">
              {{ t('redeem.cooldown', { sec: cooldownSec }) }}
            </p>
          </template>

          <div class="mt-4 flex flex-col gap-2 sm:flex-row">
            <Button
              v-if="resultKind === 'fail' && cooldownSec <= 0"
              class="font-bold text-white sm:flex-1"
              :style="accentStyle"
              @click="retryCred"
            >
              {{ t('redeem.retry') }}
            </Button>
            <Button as-child class="font-bold text-white sm:flex-1" :style="accentStyle">
              <router-link to="/">{{ t('redeem.home') }}</router-link>
            </Button>
          </div>
          <p class="mt-4 text-center text-xs text-muted-foreground">{{ t('redeem.csHint') }}</p>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  redeemAPI,
  type RedeemStatusResult,
  type RedeemVerifyResult,
} from '@/api/redeem'

const { t } = useI18n()
const route = useRoute()
const accent = '#6E56CF'
const accentStyle = { backgroundColor: accent }

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
const lastRefresh = ref<Date | null>(null)
const credHint = ref('')
const credError = ref('')

let cooldownTimer: number | undefined
let pollTimer: number | undefined

const stepLabels = computed(() => [
  t('redeem.step1'),
  t('redeem.step2'),
  t('redeem.step3'),
  t('redeem.step4'),
])

const stepIndex = computed(() => {
  if (mode.value === 'query' && step.value === 'verify') return 0
  if (step.value === 'verify') return 0
  if (step.value === 'cred') return 1
  if (step.value === 'proc') return 2
  return 3
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

const resultKind = computed<'ok' | 'fail' | 'pending'>(() => {
  const s = status.value?.status
  if (s === 'succeeded') return 'ok'
  if (s === 'pending_confirm') return 'pending'
  return 'fail'
})

const autoRenewLabel = computed(() => {
  if (status.value?.subscription_cancelled === true) return t('redeem.autoRenewOff')
  if (status.value?.subscription_cancelled === false) return t('redeem.autoRenewOn')
  return t('redeem.autoRenewUnknown')
})

const stageUpstreamLabel = computed(() => {
  const s = status.value?.status
  if (s === 'processing' || s === 'locked') return t('redeem.inProgress')
  if (s === 'pending_confirm') return t('redeem.confirming')
  return status.value?.stage || '—'
})

const lastRefreshLabel = computed(() => {
  if (!lastRefresh.value) return '—'
  return lastRefresh.value.toLocaleTimeString('zh-CN', { hour12: false })
})

function stepCircleClass(i: number) {
  if (i < stepIndex.value) return 'bg-[#6E56CF] text-white'
  if (i === stepIndex.value) return 'border-2 border-[#6E56CF] bg-transparent'
  return 'bg-muted text-muted-foreground'
}

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

function applyStatus(res: RedeemStatusResult) {
  status.value = res
  lastRefresh.value = new Date()
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
    error.value = e?.message || t('redeem.networkError')
    // If maintenance, don't clear credential so user can retry later? Design: don't persist session.
    // Keep credential in form for retry after maintenance - actually design says never store; keeping in Vue ref until leave is fine for UX on maintenance.
  } finally {
    busy.value = false
  }
}

async function fetchStatus(forceResult = false) {
  try {
    const { data } = await redeemAPI.status(normalizeCode(code.value))
    const res = (data?.data || data) as RedeemStatusResult
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

function startPoll() {
  stopPoll()
  pollTimer = window.setInterval(() => {
    fetchStatus()
  }, 5000)
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

function backToVerify() {
  mode.value = 'redeem'
  step.value = 'verify'
  error.value = ''
  stopPoll()
}

function formatCompleted(v?: string) {
  if (!v) return '—'
  try {
    const d = new Date(v)
    if (Number.isNaN(d.getTime())) return v
    return d.toLocaleString('zh-CN', { hour12: false, timeZone: 'Asia/Shanghai' }) + ' (UTC+8)'
  } catch {
    return v
  }
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
