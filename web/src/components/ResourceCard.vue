<script setup>
import { computed } from 'vue'
import NumberStepper from './NumberStepper.vue'
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

function bump(target, delta) {
  emit('change', { target, resource: props.resourceKey, delta })
}
</script>

<template>
  <div
    class="rcard rounded-2xl border border-surface-border bg-surface-raised/80 p-4 animate-fadeUp"
    :style="{ borderTopColor: meta.accent, borderTopWidth: '3px' }"
  >
    <div class="rcard-head mb-3 flex items-baseline justify-between">
      <h3 class="flex items-center gap-1.5 font-display text-sm tracking-wide" :style="{ color: meta.accent }">
        <component :is="icon" v-if="icon" class="rcard-icon h-4 w-4" />
        {{ meta.label }}
      </h3>
    </div>

    <div class="rcard-stats grid grid-cols-2 gap-3">
      <div>
        <p class="rcard-stat-label text-[10px] uppercase tracking-wider text-ink-muted">Stock</p>
        <NumberStepper
          :value="stock"
          :label="`${meta.label} Stock`"
          :disabled="!interactive"
          @delta="(d) => bump('stock', d)"
        />
      </div>
      <div>
        <p class="rcard-stat-label text-[10px] uppercase tracking-wider text-ink-muted">Production</p>
        <NumberStepper
          :value="production"
          :label="`${meta.label} Production`"
          value-class="text-ink-muted"
          show-sign
          :disabled="!interactive"
          @delta="(d) => bump('production', d)"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 横画面では6資源を2x3で一度に見せたいが、縦画面基準のパディング/フォント
   のままだと3行ぶんの高さが画面に収まらない。既存の見た目・構造は変えず、
   横画面のときだけ余白とフォントを詰めて高さを抑える。1024px以上
   (タブレット横〜PC)は対象外にし、既存レイアウトを尊重する。 */
@media (orientation: landscape) and (max-width: 1023px) {
  .rcard {
    padding: 8px 10px;
  }
  .rcard-head {
    margin-bottom: 4px;
  }
  .rcard-head h3 {
    font-size: 11px;
  }
  .rcard-icon {
    height: 12px;
    width: 12px;
  }
  .rcard-stats {
    gap: 8px;
  }
  .rcard-stat-label {
    font-size: 8px;
    margin-bottom: 1px;
  }
}
</style>
