<script setup>
import { computed } from 'vue'

const props = defineProps({
  players: { type: Array, required: true },
  playerId: { type: String, required: true },
  scoreFields: { type: Array, required: true },
})

const emit = defineEmits(['score'])

function scoreValue(player, field) {
  const s = player?.score
  if (!s || typeof s !== 'object') return 0
  const v = s[field]
  return typeof v === 'number' ? v : 0
}

function calcTotal(player) {
  if (!player) return 0
  let sum = player.tr || 0
  for (const f of props.scoreFields) {
    sum += scoreValue(player, f.field)
  }
  return sum
}

const ranked = computed(() =>
  [...props.players]
    .map((p) => ({ player: p, total: calcTotal(p) }))
    .sort((a, b) => b.total - a.total),
)
</script>

<template>
  <section class="mb-4 rounded-2xl border border-amber-500/40 bg-[#1a1620] p-4 text-ink shadow-toast sm:p-6">
    <h2 class="font-display text-lg tracking-wide text-amber-300">VP ヘルパ</h2>
    <p class="mt-1 text-sm text-ink-muted">
      TR は自動反映。緑化・都市・賞などは自分の行だけ +/- で調整できます。
    </p>

    <div class="mt-4 space-y-4">
      <article
        v-for="row in ranked"
        :key="row.player.id"
        class="rounded-xl border border-surface-border bg-surface-raised p-4"
        :class="row.player.id === playerId ? 'ring-1 ring-mars/60' : ''"
      >
        <div class="mb-3 flex flex-wrap items-end justify-between gap-2">
          <div class="flex items-center gap-2">
            <span class="h-3 w-3 rounded-full" :style="{ backgroundColor: row.player.color }" />
            <h3 class="font-semibold">{{ row.player.name }}</h3>
          </div>
          <p class="font-display text-3xl font-bold text-amber-300 tabular-nums">
            {{ row.total }}
            <span class="text-sm font-sans font-normal text-ink-muted">VP</span>
          </p>
        </div>

        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <div class="rounded-lg bg-surface px-3 py-2">
            <p class="text-[10px] uppercase tracking-wider text-ink-muted">TR</p>
            <p class="font-display text-xl tabular-nums">{{ row.player.tr }}</p>
          </div>
          <div
            v-for="f in scoreFields"
            :key="f.field"
            class="rounded-lg bg-surface px-3 py-2"
          >
            <p class="text-[10px] uppercase tracking-wider text-ink-muted">{{ f.label }}</p>
            <div class="mt-1 flex items-center gap-2">
              <p class="font-display text-xl tabular-nums">{{ scoreValue(row.player, f.field) }}</p>
              <template v-if="row.player.id === playerId">
                <button
                  type="button"
                  class="rounded-md bg-surface-border px-2 py-1 text-sm hover:bg-mars"
                  @click="emit('score', { field: f.field, delta: -1 })"
                >
                  −
                </button>
                <button
                  type="button"
                  class="rounded-md bg-surface-border px-2 py-1 text-sm hover:bg-mars"
                  @click="emit('score', { field: f.field, delta: 1 })"
                >
                  +
                </button>
              </template>
            </div>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
