<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  title: { type: String, default: '' },
  min: { type: Number, default: 0 },
  max: { type: Number, default: 99 },
})

const emit = defineEmits(['submit', 'cancel'])

const digits = ref('')

const displayValue = computed(() => (digits.value ? digits.value : '0'))
const numericValue = computed(() => Number(digits.value || 0))

function tapDigit(d) {
  if (digits.value === '' && d === 0) return // 先頭の0は無視
  const next = digits.value + String(d)
  if (Number(next) > props.max) return
  digits.value = next
}

function clearDigits() {
  digits.value = ''
}

function submit() {
  emit('submit', Math.max(props.min, numericValue.value))
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
          <p class="text-xs text-ink-muted">{{ title }}（0〜{{ max }}）</p>
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
            class="rounded-lg bg-surface py-3 text-lg font-semibold text-ink hover:bg-surface-border"
            @click="clearDigits"
          >
            C
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
            class="rounded-lg bg-cyan-700 py-3 text-lg font-semibold text-white hover:bg-cyan-600"
            @click="submit"
          >
            確定
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
