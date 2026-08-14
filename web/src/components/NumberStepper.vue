<script setup>
import { ref, computed } from 'vue'
import RepeatPressButton from './RepeatPressButton.vue'
import DeltaKeypad from './DeltaKeypad.vue'

const props = defineProps({
  value: { type: Number, required: true },
  label: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  showSign: { type: Boolean, default: false },
  valueClass: { type: String, default: '' },
})

const emit = defineEmits(['delta'])

const keypadOpen = ref(false)

const displayValue = computed(() =>
  props.showSign && props.value >= 0 ? `+${props.value}` : `${props.value}`,
)

function submitKeypad(delta) {
  keypadOpen.value = false
  emit('delta', delta)
}
</script>

<template>
  <div>
    <button
      type="button"
      class="font-display text-3xl font-bold tabular-nums leading-none disabled:pointer-events-none disabled:opacity-60"
      :class="valueClass"
      :disabled="disabled"
      title="タップして増減値を直接入力"
      @click="keypadOpen = true"
    >
      {{ displayValue }}
    </button>

    <div class="mt-2 flex gap-1">
      <RepeatPressButton
        class="flex-1 rounded-md bg-surface px-1 py-1.5 text-xs font-semibold text-ink hover:bg-surface-border"
        :disabled="disabled"
        @press="emit('delta', -1)"
      >
        −1
      </RepeatPressButton>
      <RepeatPressButton
        class="flex-1 rounded-md bg-surface px-1 py-1.5 text-xs font-semibold text-ink hover:bg-surface-border"
        :disabled="disabled"
        @press="emit('delta', 1)"
      >
        +1
      </RepeatPressButton>
    </div>

    <DeltaKeypad
      v-if="keypadOpen"
      :title="label"
      :current-value="value"
      @submit="submitKeypad"
      @cancel="keypadOpen = false"
    />
  </div>
</template>
