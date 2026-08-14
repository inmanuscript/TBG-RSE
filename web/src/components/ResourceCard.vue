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
    class="rounded-2xl border border-surface-border bg-surface-raised/80 p-4 animate-fadeUp"
    :style="{ borderTopColor: meta.accent, borderTopWidth: '3px' }"
  >
    <div class="mb-3 flex items-baseline justify-between">
      <h3 class="flex items-center gap-1.5 font-display text-sm tracking-wide" :style="{ color: meta.accent }">
        <component :is="icon" v-if="icon" class="h-4 w-4" />
        {{ meta.label }}
      </h3>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <p class="text-[10px] uppercase tracking-wider text-ink-muted">Stock</p>
        <NumberStepper
          :value="stock"
          :label="`${meta.label} Stock`"
          :disabled="!interactive"
          @delta="(d) => bump('stock', d)"
        />
      </div>
      <div>
        <p class="text-[10px] uppercase tracking-wider text-ink-muted">Production</p>
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
