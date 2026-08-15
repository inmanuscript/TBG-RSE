<script setup>
import { computed, ref } from 'vue'
import DeltaKeypad from './DeltaKeypad.vue'
import { RESOURCE_ICONS } from '../resourceIcons'

const props = defineProps({
  resourceKey: { type: String, required: true },
  meta: { type: Object, required: true },
  stock: { type: Number, required: true },
  production: { type: Number, required: true },
  interactive: { type: Boolean, default: true },
})

const emit = defineEmits(['change'])

const icon = computed(() => RESOURCE_ICONS[props.resourceKey])

// 'stock' | 'production' | null — テンキーを開いているか（初期選択）
const keypadOpen = ref(false)
const initialTarget = ref('stock')

function openKeypad(target) {
  if (!props.interactive) return
  initialTarget.value = target
  keypadOpen.value = true
}

const keypadTargetOptions = computed(() => [
  { value: 'stock', label: 'Stock', currentValue: props.stock },
  { value: 'production', label: 'Production', currentValue: props.production },
])

function submitKeypad({ target, delta }) {
  keypadOpen.value = false
  emit('change', { target, resource: props.resourceKey, delta })
}
</script>

<template>
  <!-- OpponentCardの資源セルとほぼ同一の見た目(rounded-lg bg-surface px-2 py-2 text-center)。
       上下タップは初期選択のみ。開いたテンキー内で Stock / Production を切替可能。
       ・数値下のごく薄い点線(「編集可能な値」の慣用表現)
       ・タップ時のscale-down + 背景の一瞬の明滅
       でタップ可能であることをさりげなく伝える。 -->
  <div class="rounded-lg bg-surface px-2 py-2 text-center">
    <button
      type="button"
      class="block w-full rounded-md transition duration-100 active:scale-95 active:bg-surface-border disabled:pointer-events-none disabled:opacity-60"
      :disabled="!interactive"
      title="タップして増減（Stock）"
      @click="openKeypad('stock')"
    >
      <span class="flex items-center justify-center gap-0.5 text-[10px] font-medium" :style="{ color: meta.accent }">
        <component :is="icon" v-if="icon" class="h-3 w-3" />
        {{ meta.short }}
      </span>
      <span
        class="inline-block border-b border-dashed pb-0.5 font-display text-lg leading-tight tabular-nums"
        :style="{ borderColor: meta.accent }"
      >
        {{ stock }}
      </span>
    </button>

    <button
      type="button"
      class="mt-0.5 block w-full rounded-md py-1 transition duration-100 active:scale-95 active:bg-surface-border disabled:pointer-events-none disabled:opacity-60"
      :disabled="!interactive"
      title="タップして増減（Production）"
      @click="openKeypad('production')"
    >
      <span class="inline-block border-b border-dashed border-ink-muted/40 pb-0.5 text-[10px] tabular-nums text-ink-muted">
        {{ production >= 0 ? '+' : '' }}{{ production }}
      </span>
    </button>

    <DeltaKeypad
      v-if="keypadOpen"
      :title="meta.short"
      :accent="meta.accent"
      :icon="icon"
      :target-options="keypadTargetOptions"
      :initial-target="initialTarget"
      @submit="submitKeypad"
      @cancel="keypadOpen = false"
    />
  </div>
</template>
