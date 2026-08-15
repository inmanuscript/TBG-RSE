<script setup>
import { computed } from 'vue'
import { WifiOff } from '@lucide/vue'
import { RESOURCE_ICONS } from '../resourceIcons'

const props = defineProps({
  player: { type: Object, required: true },
  resourceOrder: { type: Array, required: true },
  resourceMeta: { type: Object, required: true },
  highlighted: { type: Boolean, default: false },
  active: { type: Boolean, default: false },
  totalVp: { type: Number, default: null },
})

const emit = defineEmits(['skip'])

const resources = computed(() =>
  props.resourceOrder.map((key) => ({
    key,
    meta: props.resourceMeta[key],
    icon: RESOURCE_ICONS[key],
    stock: props.player.resources?.[key]?.stock ?? 0,
    production: props.player.resources?.[key]?.production ?? 0,
  })),
)
</script>

<template>
  <div
    class="rounded-2xl border bg-surface-raised/70 p-4 transition"
    :class="{
      'border-mars shadow-[0_0_0_1px_rgba(212,101,47,0.5)]': highlighted || active,
      'border-surface-border': !highlighted && !active,
      'opacity-60': player.passed,
    }"
  >
    <div class="mb-3 flex items-center gap-3">
      <span class="h-3.5 w-3.5 rounded-full" :style="{ backgroundColor: player.color }" />
      <div class="min-w-0 flex-1">
        <p class="truncate font-semibold">
          {{ player.name }}
          <span v-if="active" class="ml-1 text-xs text-mars-glow">TURN</span>
          <span v-if="player.passed" class="ml-1 text-xs text-red-300">PASS</span>
          <span v-if="player.online === false" class="ml-1 inline-flex items-center gap-0.5 text-xs text-amber-300">
            <WifiOff class="h-3 w-3" />
            オフライン
          </span>
        </p>
        <p class="text-xs text-ink-muted">
          TR {{ player.tr }}
          <span v-if="totalVp != null" class="ml-2 text-amber-300">VP {{ totalVp }}</span>
        </p>
      </div>
      <button
        v-if="player.online === false"
        type="button"
        class="shrink-0 rounded-lg border border-amber-700/50 bg-amber-950/30 px-2.5 py-1.5 text-xs font-medium text-amber-200 hover:border-amber-500"
        @click="emit('skip', player.id)"
      >
        スキップ
      </button>
    </div>
    <div class="grid grid-cols-3 gap-2 sm:grid-cols-6">
      <div
        v-for="r in resources"
        :key="r.key"
        class="rounded-lg bg-surface px-2 py-2 text-center"
      >
        <p class="flex items-center justify-center gap-0.5 text-[10px] font-medium" :style="{ color: r.meta.accent }">
          <component :is="r.icon" v-if="r.icon" class="h-3 w-3" />
          {{ r.meta.short }}
        </p>
        <p class="font-display text-lg tabular-nums leading-tight">{{ r.stock }}</p>
        <p class="text-[10px] tabular-nums text-ink-muted">
          {{ r.production >= 0 ? '+' : '' }}{{ r.production }}
        </p>
      </div>
    </div>
  </div>
</template>
