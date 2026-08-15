<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  title: { type: String, default: '' },
  currentValue: { type: Number, default: null },
  // 資源パネル向け。指定時はタイトルをアクセント色＋アイコンで表示
  accent: { type: String, default: null },
  icon: { type: [Object, Function], default: null },
  // 指定時はテンキー内で対象切替が可能。例: [{ value: 'stock', label: 'Stock', currentValue: 3 }, ...]
  targetOptions: { type: Array, default: null },
  initialTarget: { type: String, default: null },
})

const emit = defineEmits(['submit', 'cancel'])

const digits = ref('')
const selectedTarget = ref(
  props.initialTarget
    ?? props.targetOptions?.[0]?.value
    ?? null,
)

const hasTargetSwitch = computed(() => Array.isArray(props.targetOptions) && props.targetOptions.length > 0)

const activeCurrentValue = computed(() => {
  if (hasTargetSwitch.value) {
    const opt = props.targetOptions.find((o) => o.value === selectedTarget.value)
    return opt?.currentValue ?? null
  }
  return props.currentValue
})

const displayValue = computed(() => (digits.value ? digits.value : '0'))
const canSubmit = computed(() => digits.value.length > 0)

function selectTarget(value) {
  if (selectedTarget.value === value) return
  selectedTarget.value = value
  digits.value = ''
}

function tapDigit(d) {
  if (digits.value === '' && d === 0) return // 先頭の0は無視
  if (digits.value.length >= 4) return
  digits.value += String(d)
}

function clearDigits() {
  digits.value = ''
}

function submit(sign) {
  if (!canSubmit.value) return
  const delta = sign * Number(digits.value)
  if (hasTargetSwitch.value) {
    emit('submit', { target: selectedTarget.value, delta })
  } else {
    emit('submit', delta)
  }
}
</script>

<template>
  <Teleport to="body">
    <div
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4"
      @click.self="$emit('cancel')"
    >
      <div class="w-full max-w-[280px] rounded-2xl border border-surface-border bg-surface-raised p-4 shadow-toast">
        <div class="mb-3 flex items-center justify-between gap-2">
          <p class="flex min-w-0 items-center gap-1.5 font-display text-base font-semibold">
            <component
              :is="icon"
              v-if="icon"
              class="h-4 w-4 shrink-0"
              :style="accent ? { color: accent } : undefined"
            />
            <span class="truncate" :style="accent ? { color: accent } : undefined" :class="accent ? '' : 'text-ink'">
              {{ title }}
            </span>
            <span
              v-if="activeCurrentValue !== null"
              class="shrink-0 text-sm font-normal tabular-nums text-ink-muted"
            >
              （現在 {{ activeCurrentValue }}）
            </span>
          </p>
          <button
            type="button"
            class="shrink-0 rounded-md px-1.5 py-0.5 font-display text-sm text-ink-muted hover:bg-surface hover:text-ink"
            aria-label="閉じる"
            @click="$emit('cancel')"
          >
            ×
          </button>
        </div>

        <div
          v-if="hasTargetSwitch"
          class="mb-3 grid grid-cols-2 gap-1 rounded-xl bg-surface p-1"
          role="tablist"
          aria-label="編集対象"
        >
          <button
            v-for="opt in targetOptions"
            :key="opt.value"
            type="button"
            role="tab"
            class="rounded-lg py-2.5 font-display text-sm font-semibold transition"
            :class="selectedTarget === opt.value
              ? 'bg-surface-raised text-ink shadow-sm'
              : 'text-ink-muted hover:text-ink'"
            :aria-selected="selectedTarget === opt.value"
            @click="selectTarget(opt.value)"
          >
            {{ opt.label }}
          </button>
        </div>

        <button
          type="button"
          class="mb-3 w-full rounded-xl bg-surface px-3 py-3 text-center font-display text-3xl font-bold tabular-nums hover:bg-surface-border"
          title="タップでクリア"
          @click="clearDigits"
        >
          {{ displayValue }}
        </button>

        <div class="grid grid-cols-3 gap-1.5">
          <button
            v-for="n in [1, 2, 3, 4, 5, 6, 7, 8, 9]"
            :key="n"
            type="button"
            class="rounded-lg bg-surface py-3 font-display text-lg font-semibold tabular-nums text-ink hover:bg-surface-border"
            @click="tapDigit(n)"
          >
            {{ n }}
          </button>
          <button
            type="button"
            class="rounded-lg py-3 font-display text-lg font-semibold disabled:opacity-30"
            :class="canSubmit ? 'bg-red-950/50 text-red-200 hover:bg-red-900/50' : 'bg-surface text-ink-muted'"
            :disabled="!canSubmit"
            @click="submit(-1)"
          >
            −
          </button>
          <button
            type="button"
            class="rounded-lg bg-surface py-3 font-display text-lg font-semibold tabular-nums text-ink hover:bg-surface-border"
            @click="tapDigit(0)"
          >
            0
          </button>
          <button
            type="button"
            class="rounded-lg py-3 font-display text-lg font-semibold disabled:opacity-30"
            :class="canSubmit ? 'bg-emerald-950/50 text-emerald-200 hover:bg-emerald-900/50' : 'bg-surface text-ink-muted'"
            :disabled="!canSubmit"
            @click="submit(1)"
          >
            +
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
