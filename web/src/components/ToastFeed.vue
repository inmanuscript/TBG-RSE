<script setup>
import { AlertCircle, X } from '@lucide/vue'

defineProps({
  toasts: { type: Array, required: true },
})

const emit = defineEmits(['dismiss'])
</script>

<template>
  <div class="pointer-events-none fixed top-3 left-1/2 -translate-x-1/2 sm:left-auto sm:right-3 sm:translate-x-0 z-50 flex w-[min(100%-1.5rem,24rem)] flex-col gap-2">
    <div
      v-for="t in toasts"
      :key="t.id"
      class="pointer-events-auto flex items-start justify-between gap-2.5 animate-slideIn rounded-xl border px-4 py-3 shadow-toast backdrop-blur transition-all"
      :class="t.isError
        ? 'border-red-500/60 bg-red-950/95 text-red-200 shadow-red-950/40'
        : 'border-surface-border bg-surface-raised/95 text-ink'"
    >
      <div class="flex items-start gap-2.5 min-w-0">
        <AlertCircle v-if="t.isError" class="mt-0.5 h-4 w-4 shrink-0 text-red-400" />
        <div class="min-w-0">
          <p v-if="!t.isError" class="text-xs text-ink-muted">{{ t.timestamp }} · {{ t.playerName }}</p>
          <p v-else class="text-xs font-bold text-red-300">エラー</p>
          <p class="mt-0.5 text-sm" :class="t.isError ? 'text-red-100 font-medium' : 'text-ink'">{{ t.message }}</p>
        </div>
      </div>
      <button
        type="button"
        class="shrink-0 rounded p-1 transition"
        :class="t.isError ? 'text-red-300 hover:bg-red-900/50 hover:text-white' : 'text-ink-muted hover:bg-surface hover:text-ink'"
        title="閉じる"
        @click="emit('dismiss', t.id)"
      >
        <X class="h-4 w-4" />
      </button>
    </div>
  </div>
</template>
