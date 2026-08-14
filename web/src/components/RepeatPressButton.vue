<script setup>
import { useRepeatPress } from '../composables/useRepeatPress'

const emit = defineEmits(['press'])

defineOptions({ inheritAttrs: false })

const handlers = useRepeatPress(() => emit('press'))
</script>

<template>
  <button type="button" v-bind="$attrs" v-on="handlers">
    <slot />
  </button>
</template>
