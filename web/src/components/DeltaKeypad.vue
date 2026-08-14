<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  title: { type: String, default: '' },
  currentValue: { type: Number, default: null },
})

const emit = defineEmits(['submit', 'cancel'])

const digits = ref('')

const displayValue = computed(() => (digits.value ? digits.value : '0'))
const canSubmit = computed(() => digits.value.length > 0)

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
  emit('submit', sign * Number(digits.value))
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
          <p class="text-xs text-ink-muted">
            {{ title }}<span v-if="currentValue !== null">（現在 {{ currentValue }}）</span>を増減
          </p>
          <button
            type="button"
            class="rounded-md px-1.5 py-0.5 text-sm text-ink-muted hover:bg-surface hover:text-ink"
            aria-label="閉じる"
            @click="$emit('cancel')"
          >
            ×
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
            class="rounded-lg bg-surface py-3 text-lg font-semibold text-ink hover:bg-surface-border"
            @click="tapDigit(n)"
          >
            {{ n }}
          </button>
          <button
            type="button"
            class="rounded-lg py-3 text-lg font-semibold disabled:opacity-30"
            :class="canSubmit ? 'bg-red-950/50 text-red-200 hover:bg-red-900/50' : 'bg-surface text-ink-muted'"
            :disabled="!canSubmit"
            @click="submit(-1)"
          >
            −
          </button>
          <button
            type="button"
            class="rounded-lg bg-surface py-3 text-lg font-semibold text-ink hover:bg-surface-border"
            @click="tapDigit(0)"
          >
            0
          </button>
          <button
            type="button"
            class="rounded-lg py-3 text-lg font-semibold disabled:opacity-30"
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
