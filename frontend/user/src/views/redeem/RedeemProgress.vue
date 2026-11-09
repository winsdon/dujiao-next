<template>
  <div>
    <!-- Hero -->
    <div class="flex flex-col items-center text-center">
      <div class="relative flex size-20 items-center justify-center">
        <span
          class="redeem-halo absolute inset-0 rounded-full motion-safe:animate-[spin_2.4s_linear_infinite]"
          :class="phase === 'captcha' ? 'text-warning' : 'text-primary'"
          aria-hidden="true"
        />
        <span
          class="flex size-16 items-center justify-center rounded-full"
          :class="phase === 'captcha' ? 'bg-warning-soft text-warning' : 'bg-primary-soft text-primary'"
        >
          <component :is="heroIcon" class="size-7" />
        </span>
      </div>
      <h1 class="mt-4 text-xl font-bold">{{ heroTitle }}</h1>
      <p class="mt-1.5 text-sm text-muted-foreground">
        <span class="font-mono tabular-nums text-foreground">{{ t('redeem.elapsed', { time: elapsedLabel }) }}</span>
        <span class="mx-1.5">·</span>
        <span>{{ t('redeem.typicalTime') }}</span>
      </p>
      <div class="mt-4 h-1.5 w-full overflow-hidden rounded-full bg-muted" aria-hidden="true">
        <div
          class="h-full rounded-full"
          :class="phase === 'captcha' ? 'w-full bg-warning/50' : 'redeem-shimmer w-1/3 bg-primary'"
        />
      </div>
      <p v-if="isSlow && phase !== 'captcha'" class="mt-2 text-xs text-warning">{{ t('redeem.slowHint') }}</p>
    </div>

    <!-- Timeline -->
    <ol class="mt-6 border-t pt-5">
      <li v-for="(node, i) in nodes" :key="node.key" class="relative flex gap-3 pb-5 last:pb-0">
        <span
          v-if="i < nodes.length - 1"
          class="absolute top-7 bottom-1 left-[11px] w-0.5 rounded-full transition-colors duration-500"
          :class="node.state === 'done' ? 'bg-success/60' : 'bg-border'"
          aria-hidden="true"
        />
        <span class="relative mt-0.5 flex size-6 shrink-0 items-center justify-center">
          <span
            v-if="node.state === 'active' || node.state === 'attention'"
            class="absolute inset-0 rounded-full opacity-40 motion-safe:animate-ping"
            :class="node.state === 'attention' ? 'bg-warning' : 'bg-primary'"
            aria-hidden="true"
          />
          <span class="relative flex size-6 items-center justify-center rounded-full" :class="markerClass(node.state)">
            <Check v-if="node.state === 'done'" class="size-3.5" stroke-width="3" />
            <Loader2 v-else-if="node.state === 'active'" class="size-3.5 motion-safe:animate-spin" />
            <ShieldAlert v-else-if="node.state === 'attention'" class="size-3.5" />
            <Pause v-else-if="node.state === 'paused'" class="size-3" />
            <span v-else class="size-1.5 rounded-full bg-muted-foreground/50" />
          </span>
        </span>

        <div class="min-w-0 flex-1">
          <div class="flex items-center justify-between gap-3">
            <span
              class="text-sm"
              :class="node.state === 'idle' ? 'text-muted-foreground' : 'font-medium text-foreground'"
            >
              {{ node.title }}
            </span>
            <span class="shrink-0 text-xs tabular-nums" :class="labelClass(node.state)">{{ node.label }}</span>
          </div>
          <p v-if="node.detail" class="mt-0.5 text-xs text-muted-foreground">{{ node.detail }}</p>

          <div
            v-if="node.key === 'captcha' && node.state === 'attention'"
            class="mt-3 rounded-xl border border-warning/40 bg-warning-soft p-4"
          >
            <div class="flex items-center gap-1.5 text-sm font-semibold">
              <ShieldAlert class="size-4 text-warning" />
              {{ t('redeem.captchaTitle') }}
            </div>
            <p class="mt-1 text-xs text-muted-foreground">{{ t('redeem.captchaHint') }}</p>
            <Button size="sm" class="mt-3 font-semibold" @click="emit('open-captcha')">
              {{ t('redeem.captchaBtn') }}
              <ExternalLink class="size-3.5" />
            </Button>
          </div>
        </div>
      </li>
    </ol>

    <!-- Footer -->
    <div class="mt-6 rounded-xl bg-muted/50 px-3 py-2.5">
      <div class="flex items-center justify-between gap-2 text-xs">
        <span class="flex items-center gap-2 text-muted-foreground">
          <span class="relative flex size-2" aria-hidden="true">
            <span class="absolute inline-flex size-full rounded-full bg-success opacity-60 motion-safe:animate-ping" />
            <span class="relative inline-flex size-2 rounded-full bg-success" />
          </span>
          <span aria-live="polite">
            {{ refreshing ? t('redeem.refreshing') : t('redeem.autoRefresh', { sec: nextRefreshIn }) }}
          </span>
        </span>
        <Button
          variant="ghost"
          size="sm"
          class="h-7 px-2 text-xs"
          :disabled="refreshing"
          @click="emit('refresh')"
        >
          <RefreshCw class="size-3.5" :class="refreshing ? 'motion-safe:animate-spin' : ''" />
          {{ t('redeem.refreshNow') }}
        </Button>
      </div>
      <div class="mt-1.5 flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-xs text-muted-foreground">
        <span v-if="maskedCode">
          {{ t('redeem.code') }} <span class="font-mono text-foreground">{{ maskedCode }}</span>
        </span>
        <span>{{ t('redeem.procHint') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Clock, ExternalLink, Loader2, Pause, RefreshCw, ShieldAlert, Zap } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import type { RedeemStatusResult } from '@/api/redeem'
import {
  REDEEM_SLOW_THRESHOLD_SEC,
  formatElapsed,
  parseCstDateTime,
  resolveRedeemPhase,
  resolveTimelineStates,
  type RedeemNodeState,
} from '@/utils/redeemProgress'

const props = defineProps<{
  status: RedeemStatusResult | null
  productName?: string
  maskedCode?: string
  nextRefreshIn: number
  refreshing: boolean
  /** 拿不到 submitted_at 时用来计算已等待时长的兜底起点（毫秒） */
  fallbackStartedAt: number
}>()

const emit = defineEmits<{
  refresh: []
  'open-captcha': []
}>()

const { t, locale } = useI18n()

const now = ref(Date.now())
let clock: number | undefined

onMounted(() => {
  clock = window.setInterval(() => {
    now.value = Date.now()
  }, 1000)
})

onBeforeUnmount(() => {
  if (clock) window.clearInterval(clock)
})

const phase = computed(() => resolveRedeemPhase(props.status?.status, Boolean(props.status?.captcha?.url)))
const states = computed(() => resolveTimelineStates(phase.value))

const startedAt = computed(() => parseCstDateTime(props.status?.submitted_at) ?? props.fallbackStartedAt)
const elapsedSec = computed(() => Math.max(0, (now.value - startedAt.value) / 1000))
const elapsedLabel = computed(() => formatElapsed(elapsedSec.value))
const isSlow = computed(() => elapsedSec.value > REDEEM_SLOW_THRESHOLD_SEC)

const heroIcon = computed(() => {
  if (phase.value === 'captcha') return ShieldAlert
  if (phase.value === 'confirming') return Clock
  return Zap
})

const heroTitle = computed(() => {
  if (phase.value === 'captcha') return t('redeem.procTitleCaptcha')
  if (phase.value === 'confirming') return t('redeem.procTitleConfirming')
  return props.productName ? t('redeem.procTitleNamed', { name: props.productName }) : t('redeem.procTitle')
})

const submittedTimeLabel = computed(() =>
  new Date(startedAt.value).toLocaleTimeString(locale.value, { hour12: false }),
)

type TimelineNode = {
  key: 'submitted' | 'processing' | 'captcha' | 'confirm'
  title: string
  state: RedeemNodeState
  label: string
  detail?: string
}

const nodes = computed<TimelineNode[]>(() => {
  const s = states.value
  const message = props.status?.message || ''
  return [
    {
      key: 'submitted',
      title: t('redeem.stageSubmitted'),
      state: s.submitted,
      label: submittedTimeLabel.value,
    },
    {
      key: 'processing',
      title: t('redeem.stageProcessing'),
      state: s.processing,
      label: s.processing === 'active' ? t('redeem.inProgress') : s.processing === 'paused' ? t('redeem.paused') : '',
      detail: s.processing === 'active' ? message : undefined,
    },
    {
      key: 'captcha',
      title: t('redeem.stageCaptcha'),
      state: s.captcha,
      label: s.captcha === 'attention' ? t('redeem.captchaNeeded') : t('redeem.noActionNeeded'),
    },
    {
      key: 'confirm',
      title: t('redeem.stageConfirm'),
      state: s.confirm,
      label: s.confirm === 'active' ? t('redeem.confirming') : t('redeem.waiting'),
      detail: s.confirm === 'active' ? message : undefined,
    },
  ]
})

function markerClass(state: RedeemNodeState) {
  switch (state) {
    case 'done':
      return 'bg-success text-white'
    case 'active':
      return 'border-2 border-primary bg-background text-primary'
    case 'attention':
      return 'border-2 border-warning bg-warning-soft text-warning'
    case 'paused':
      return 'border border-primary/40 bg-primary-soft text-primary'
    default:
      return 'border border-border bg-muted'
  }
}

function labelClass(state: RedeemNodeState) {
  switch (state) {
    case 'active':
      return 'font-medium text-primary'
    case 'attention':
      return 'font-medium text-warning'
    default:
      return 'text-muted-foreground'
  }
}
</script>

<style scoped>
.redeem-halo {
  background: conic-gradient(from 0deg, transparent 0deg, currentColor 300deg, transparent 360deg);
  -webkit-mask: radial-gradient(farthest-side, transparent calc(100% - 3px), #000 calc(100% - 2px));
  mask: radial-gradient(farthest-side, transparent calc(100% - 3px), #000 calc(100% - 2px));
  opacity: 0.85;
}

.redeem-shimmer {
  animation: redeem-shimmer 1.6s ease-in-out infinite;
}

@keyframes redeem-shimmer {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(300%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .redeem-shimmer {
    animation: none;
    width: 100%;
    opacity: 0.4;
  }
}
</style>
