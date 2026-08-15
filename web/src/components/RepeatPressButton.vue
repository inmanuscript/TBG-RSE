<script setup>
import { useRepeatPress } from '../composables/useRepeatPress'

const emit = defineEmits(['press'])

defineOptions({ inheritAttrs: false })

const { onPointerDown, onPointerMove, onPointerUp, onClick } = useRepeatPress(() => emit('press'))
</script>

<template>
  <button
    type="button"
    v-bind="$attrs"
    @pointerdown="onPointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointerleave="onPointerUp"
    @pointercancel="onPointerUp"
    @click="onClick"
  >
    <slot />
  </button>
</template>
