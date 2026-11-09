<template>
  <div>
    <div class="flex flex-col items-center text-center">
      <span class="flex size-16 items-center justify-center rounded-full" :class="tone.circle">
        <component :is="tone.icon" class="size-8" stroke-width="2.5" />
      </span>
      <h1 class="mt-4 text-xl font-bold">{{ tone.title }}</h1>
      <p v-if="kind === 'ok' && productName" class="mt-1.5 text-sm text-muted-foreground">
        {{ t('redeem.okSubtitle', { name: productName }) }}
      </p>
      <p v-else-if="kind !== 'ok'" class="mt-2 max-w-sm text-sm text-muted-foreground">{{ message }}</p>
    </div>

    <!-- Success details -->
    <template v-if="kind === 'ok'">
      <dl class="mt-6 divide-y rounded-xl border bg-muted/30 text-sm">
        <div v-for="row in detailRows" :key="row.label" class="flex items-center justify-between gap-4 px-4 py-2.5">
          <dt class="shrink-0 text-muted-foreground">{{ row.label }}</dt>
          <dd class="min-w-0 break-words text-right" :class="row.mono ? 'font-mono' : ''">
            <span v-if="row.dot" class="mr-1.5 inline-block size-1.5 rounded-full align-middle" :class="row.dot" />
            <span :class="row.valueClass">{{ row.value }}</span>
          </dd>
        </div>
      </dl>
      <div
        v-if="status?.subscription_cancelled !== true"
        class="mt-3 flex gap-2 rounded-xl border border-warning/40 bg-warning-soft p-3 text-xs leading-relaxed text-foreground"
      >
        <AlertTriangle class="mt-0.5 size-4 shrink-0 text-warning" />
        <span>{{ t('redeem.autoRenewHint') }}</span>
      </div>
    </template>

    <!-- Failure: code not consumed -->
    <div
      v-else-if="kind === 'fail'"
      class="mt-5 flex items-center justify-center gap-1.5 rounded-xl bg-success-soft px-3 py-2 text-xs text-success"
    >
      <ShieldCheck class="size-4" />
      {{ t('redeem.codeNotConsumed') }}
    </div>

    <div class="mt-6 flex flex-col gap-2 sm:flex-row">
      <Button
        v-if="kind === 'fail'"
        class="h-11 font-semibold sm:flex-1"
        :disabled="cooldownSec > 0"
        @click="emit('retry')"
      >
        <template v-if="cooldownSec > 0">
          <Clock />
          <span class="tabular-nums">{{ t('redeem.cooldownBtn', { sec: cooldownSec }) }}</span>
        </template>
        <template v-else>
          <RotateCcw />
          {{ t('redeem.retry') }}
        </template>
      </Button>
      <Button as-child :variant="kind === 'fail' ? 'outline' : 'default'" class="h-11 font-semibold sm:flex-1">
        <router-link to="/">{{ t('redeem.home') }}</router-link>
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, Check, Clock, RotateCcw, ShieldCheck, X } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import type { RedeemStatusResult, RedeemVerifyResult } from '@/api/redeem'
import { parseCstDateTime } from '@/utils/redeemProgress'

const props = defineProps<{
  kind: 'ok' | 'fail' | 'pending'
  status: RedeemStatusResult | null
  verifyInfo: RedeemVerifyResult | null
  cooldownSec: number
  error?: string
}>()

const emit = defineEmits<{ retry: [] }>()

const { t } = useI18n()

const productName = computed(() => props.status?.product_name || props.verifyInfo?.product_name || '')

const message = computed(() => {
  if (props.kind === 'pending') return props.status?.message || t('redeem.pendingHint')
  return props.status?.message || props.error || t('redeem.failHint')
})

const tone = computed(() => {
  if (props.kind === 'ok') {
    return { icon: Check, circle: 'bg-success-soft text-success', title: t('redeem.okTitle') }
  }
  if (props.kind === 'pending') {
    return { icon: Clock, circle: 'bg-warning-soft text-warning', title: t('redeem.pendingTitle') }
  }
  return { icon: X, circle: 'bg-danger-soft text-danger', title: t('redeem.failTitle') }
})

function formatCompleted(v?: string) {
  if (!v) return '—'
  const ms = parseCstDateTime(v)
  if (ms === null) return v
  return new Date(ms).toLocaleString('zh-CN', { hour12: false, timeZone: 'Asia/Shanghai' }) + ' (UTC+8)'
}

type DetailRow = { label: string; value: string; mono?: boolean; dot?: string; valueClass?: string }

const detailRows = computed<DetailRow[]>(() => {
  const s = props.status
  const cancelled = s?.subscription_cancelled
  const renew: Pick<DetailRow, 'value' | 'dot' | 'valueClass'> =
    cancelled === true
      ? { value: t('redeem.autoRenewOff'), dot: 'bg-success', valueClass: 'text-success' }
      : cancelled === false
        ? { value: t('redeem.autoRenewOn'), dot: 'bg-warning', valueClass: 'text-warning' }
        : { value: t('redeem.autoRenewUnknown'), dot: 'bg-warning', valueClass: 'text-warning' }
  return [
    { label: t('redeem.account'), value: s?.masked_email || t('redeem.emailPending'), mono: true },
    { label: t('redeem.region'), value: s?.payment_region || '—', mono: true },
    { label: t('redeem.completedAt'), value: formatCompleted(s?.completed_at), mono: true },
    { label: t('redeem.autoRenew'), ...renew },
    {
      label: t('redeem.code'),
      value: `${s?.masked_code || props.verifyInfo?.masked_code || ''} ${t('redeem.used')}`.trim(),
      mono: true,
    },
  ]
})
</script>
