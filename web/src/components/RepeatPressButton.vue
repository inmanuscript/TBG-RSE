<script setup>
import { useRepeatPress } from '../composables/useRepeatPress'

const emit = defineEmits(['press'])

defineOptions({ inheritAttrs: false })

const { onPointerDown, onPointerUp, onClick } = useRepeatPress(() => emit('press'))
</script>

<template>
  <button
    type="button"
    v-bind="$attrs"
    @pointerdown="onPointerDown"
    @pointerup="onPointerUp"
    @pointerleave="onPointerUp"
    @pointercancel="onPointerUp"
    @click="onClick"
  >
    <slot />
  </button>
</template>
