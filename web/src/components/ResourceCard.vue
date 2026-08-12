<script setup>
const props = defineProps({
  resourceKey: { type: String, required: true },
  meta: { type: Object, required: true },
  stock: { type: Number, required: true },
  production: { type: Number, required: true },
})

const emit = defineEmits(['change'])

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
      <h3 class="font-display text-sm tracking-wide" :style="{ color: meta.accent }">
        {{ meta.label }}
      </h3>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <p class="text-[10px] uppercase tracking-wider text-ink-muted">Stock</p>
        <p class="font-display text-3xl font-bold tabular-nums leading-none">{{ stock }}</p>
        <div class="mt-2 grid grid-cols-4 gap-1">
          <button
            v-for="d in [-5, -1, 1, 5]"
            :key="'s' + d"
            type="button"
            class="rounded-md bg-surface px-1 py-1.5 text-xs font-semibold text-ink hover:bg-surface-border"
            @click="bump('stock', d)"
          >
            {{ d > 0 ? '+' : '' }}{{ d }}
          </button>
        </div>
      </div>
      <div>
        <p class="text-[10px] uppercase tracking-wider text-ink-muted">Production</p>
        <p class="font-display text-3xl font-bold tabular-nums leading-none text-ink-muted">
          {{ production >= 0 ? '+' : '' }}{{ production }}
        </p>
        <div class="mt-2 grid grid-cols-4 gap-1">
          <button
            v-for="d in [-5, -1, 1, 5]"
            :key="'p' + d"
            type="button"
            class="rounded-md bg-surface px-1 py-1.5 text-xs font-semibold text-ink hover:bg-surface-border"
            @click="bump('production', d)"
          >
            {{ d > 0 ? '+' : '' }}{{ d }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
