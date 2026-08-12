<script setup>
defineProps({
  toasts: { type: Array, required: true },
})
</script>

<template>
  <div class="pointer-events-none fixed right-3 top-3 z-50 flex w-[min(100%-1.5rem,22rem)] flex-col gap-2">
    <div
      v-for="t in toasts"
      :key="t.id"
      class="pointer-events-auto animate-slideIn rounded-xl border border-surface-border bg-surface-raised/95 px-4 py-3 shadow-toast backdrop-blur"
    >
      <p class="text-xs text-ink-muted">{{ t.timestamp }} · {{ t.playerName }}</p>
      <p class="mt-0.5 text-sm text-ink">{{ t.message }}</p>
    </div>
  </div>
</template>
