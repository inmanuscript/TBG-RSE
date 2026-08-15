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

// 'stock' | 'production' | null — どちらのテンキーを開いているか
const keypadTarget = ref(null)

function openKeypad(target) {
  if (!props.interactive) return
  keypadTarget.value = target
}

const keypadTitle = computed(() =>
  `${props.meta.label} ${keypadTarget.value === 'production' ? 'Production' : 'Stock'}`,
)
const keypadCurrentValue = computed(() =>
  keypadTarget.value === 'production' ? props.production : props.stock,
)

function submitKeypad(delta) {
  const target = keypadTarget.value
  keypadTarget.value = null
  emit('change', { target, resource: props.resourceKey, delta })
}
</script>

<template>
  <!-- OpponentCardの資源セルとほぼ同一の見た目(rounded-lg bg-surface px-2 py-2 text-center)。
       ボタン枠やピルは付けず、代わりに
       ・数値下のごく薄い点線(「編集可能な値」の慣用表現)
       ・タップ時のscale-down + 背景の一瞬の明滅
       でタップ可能であることをさりげなく伝える。 -->
  <div class="rounded-lg bg-surface px-2 py-2 text-center">
    <button
      type="button"
      class="block w-full rounded-md transition duration-100 active:scale-95 active:bg-surface-border disabled:pointer-events-none disabled:opacity-60"
      :disabled="!interactive"
      title="タップしてStockを増減"
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
      class="block w-full rounded-md transition duration-100 active:scale-95 active:bg-surface-border disabled:pointer-events-none disabled:opacity-60"
      :disabled="!interactive"
      title="タップしてProductionを増減"
      @click="openKeypad('production')"
    >
      <span class="inline-block border-b border-dashed border-ink-muted/40 pb-0.5 text-[10px] tabular-nums text-ink-muted">
        {{ production >= 0 ? '+' : '' }}{{ production }}
      </span>
    </button>

    <DeltaKeypad
      v-if="keypadTarget"
      :title="keypadTitle"
      :current-value="keypadCurrentValue"
      @submit="submitKeypad"
      @cancel="keypadTarget = null"
    />
  </div>
</template>
