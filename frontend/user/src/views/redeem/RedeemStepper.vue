<template>
  <ol class="mb-8 flex items-start" :aria-label="ariaLabel">
    <li
      v-for="(label, i) in labels"
      :key="label"
      class="relative flex flex-1 flex-col items-center text-center"
      :aria-current="i === labelIndex ? 'step' : undefined"
    >
      <!-- connector to previous step -->
      <span
        v-if="i > 0"
        class="absolute top-[15px] right-1/2 h-0.5 w-full -translate-y-1/2 rounded-full transition-colors duration-500"
        :class="i <= current ? 'bg-primary' : 'bg-border'"
        aria-hidden="true"
      />
      <span
        class="relative z-10 inline-flex size-[30px] items-center justify-center rounded-full text-xs font-semibold transition-all duration-300"
        :class="circleClass(i)"
      >
        <Check v-if="i < current" class="size-4" stroke-width="3" />
        <span v-else>{{ i + 1 }}</span>
      </span>
      <span
        class="mt-2 text-xs transition-colors"
        :class="[
          i === labelIndex ? 'font-semibold text-foreground' : 'hidden text-muted-foreground sm:block',
          i < current ? 'sm:text-foreground' : '',
        ]"
      >
        {{ label }}
      </span>
    </li>
  </ol>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Check } from 'lucide-vue-next'

const props = defineProps<{
  labels: string[]
  current: number
  ariaLabel?: string
}>()

// current 等于 labels.length 表示全部完成，此时高亮最后一步的文字
const labelIndex = computed(() => Math.min(props.current, props.labels.length - 1))

function circleClass(i: number) {
  if (i < props.current) return 'bg-primary text-primary-foreground shadow-sm'
  if (i === props.current) {
    return 'border-2 border-primary bg-background text-primary ring-4 ring-primary/15'
  }
  return 'border border-border bg-muted text-muted-foreground'
}
</script>
