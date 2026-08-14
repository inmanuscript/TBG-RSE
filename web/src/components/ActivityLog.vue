<script setup>
import { computed, ref } from 'vue'
import { ScrollText, X } from '@lucide/vue'

const props = defineProps({
  open: { type: Boolean, required: true },
  items: { type: Array, required: true },
  players: { type: Array, default: () => [] },
  resourceMeta: { type: Object, default: () => ({}) },
  myPlayerId: { type: String, default: '' },
})

defineEmits(['close'])

const filterName = ref('all')

const colorByName = computed(() => {
  const map = {}
  for (const p of props.players) map[p.name] = p.color
  return map
})

const myName = computed(() => props.players.find((p) => p.id === props.myPlayerId)?.name || '')

const playerNames = computed(() => {
  const seen = new Set()
  const names = []
  for (const item of props.items) {
    if (item.playerName && item.playerName !== 'System' && !seen.has(item.playerName)) {
      seen.add(item.playerName)
      names.push(item.playerName)
    }
  }
  return names
})

const filteredItems = computed(() => {
  if (filterName.value === 'all') return props.items
  if (filterName.value === 'me') return props.items.filter((item) => item.playerName === myName.value)
  return props.items.filter((item) => item.playerName === filterName.value)
})

function deltaClass(delta) {
  if (delta > 0) return 'text-emerald-400'
  if (delta < 0) return 'text-red-400'
  return 'text-ink-muted'
}

function formatDelta(delta) {
  return delta >= 0 ? `+${delta}` : `${delta}`
}

function kindLabel(item) {
  switch (item.kind) {
    case 'resource':
      return props.resourceMeta[item.resource]?.short || item.resource
    case 'tr':
      return 'TR'
    case 'tag':
      return item.tag
    case 'score':
      return item.field
    default:
      return ''
  }
}

function kindColor(item) {
  return item.kind === 'resource' ? props.resourceMeta[item.resource]?.accent : null
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-40 flex justify-end bg-black/40" @click.self="$emit('close')">
      <aside class="flex h-full w-full max-w-md flex-col border-l border-surface-border bg-surface shadow-toast animate-slideIn">
        <div class="flex items-center justify-between border-b border-surface-border px-4 py-3">
          <div class="flex items-center gap-2 font-display text-sm tracking-wide">
            <ScrollText class="h-4 w-4 text-mars" />
            Activity
          </div>
          <button type="button" class="rounded-lg p-2 hover:bg-surface-raised" @click="$emit('close')">
            <X class="h-4 w-4" />
          </button>
        </div>

        <div v-if="playerNames.length" class="flex flex-wrap gap-1.5 border-b border-surface-border px-4 py-2.5">
          <button
            type="button"
            class="rounded-full px-2.5 py-1 text-[11px] font-medium transition"
            :class="filterName === 'all' ? 'bg-mars text-white' : 'bg-surface-raised text-ink-muted hover:text-ink'"
            @click="filterName = 'all'"
          >
            全員
          </button>
          <button
            v-if="myName"
            type="button"
            class="rounded-full px-2.5 py-1 text-[11px] font-medium transition"
            :class="filterName === 'me' ? 'bg-mars text-white' : 'bg-surface-raised text-ink-muted hover:text-ink'"
            @click="filterName = 'me'"
          >
            自分
          </button>
          <button
            v-for="name in playerNames"
            :key="name"
            type="button"
            class="flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium transition"
            :class="filterName === name ? 'bg-mars text-white' : 'bg-surface-raised text-ink-muted hover:text-ink'"
            @click="filterName = name"
          >
            <span class="h-2 w-2 shrink-0 rounded-full" :style="{ backgroundColor: colorByName[name] || '#888' }" />
            {{ name }}
          </button>
        </div>

        <ul class="flex-1 space-y-2 overflow-y-auto p-4">
          <li
            v-for="item in filteredItems"
            :key="item.id"
            class="rounded-xl border border-surface-border bg-surface-raised/60 px-3 py-2"
          >
            <div class="flex items-center justify-between gap-2">
              <p class="flex min-w-0 items-center gap-1.5 text-[11px] text-ink-muted">
                <span
                  v-if="colorByName[item.playerName]"
                  class="h-2 w-2 shrink-0 rounded-full"
                  :style="{ backgroundColor: colorByName[item.playerName] }"
                />
                <span class="truncate">{{ item.timestamp }} · {{ item.playerName }}</span>
              </p>
              <span
                v-if="kindLabel(item)"
                class="shrink-0 rounded-md px-1.5 py-0.5 text-[10px] font-semibold"
                :class="!kindColor(item) && 'bg-surface text-ink-muted'"
                :style="kindColor(item) ? { color: kindColor(item), backgroundColor: kindColor(item) + '22' } : {}"
              >
                {{ kindLabel(item) }}
              </span>
            </div>
            <p class="text-sm">
              <template v-if="item.delta != null">
                <span class="font-display tabular-nums" :class="deltaClass(item.delta)">{{ formatDelta(item.delta) }}</span>
                <span v-if="item.finalValue != null" class="ml-1 text-ink-muted">→ {{ item.finalValue }}</span>
              </template>
              <template v-else>{{ item.message }}</template>
            </p>
          </li>
          <li v-if="!filteredItems.length" class="py-8 text-center text-sm text-ink-muted">
            {{ items.length ? '該当するログがありません' : 'まだログがありません' }}
          </li>
        </ul>
      </aside>
    </div>
  </Teleport>
</template>
