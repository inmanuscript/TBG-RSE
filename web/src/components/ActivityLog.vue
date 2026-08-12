<script setup>
import { ScrollText, X } from '@lucide/vue'

defineProps({
  open: { type: Boolean, required: true },
  items: { type: Array, required: true },
})

defineEmits(['close'])
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
        <ul class="flex-1 space-y-2 overflow-y-auto p-4">
          <li
            v-for="item in items"
            :key="item.id"
            class="rounded-xl border border-surface-border bg-surface-raised/60 px-3 py-2"
          >
            <p class="text-[11px] text-ink-muted">{{ item.timestamp }} · {{ item.playerName }}</p>
            <p class="text-sm">{{ item.message }}</p>
          </li>
          <li v-if="!items.length" class="py-8 text-center text-sm text-ink-muted">まだログがありません</li>
        </ul>
      </aside>
    </div>
  </Teleport>
</template>
