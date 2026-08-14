<script setup>
import { ref, watch, computed } from 'vue'
import { X, Sliders, Check, RotateCcw, Thermometer, Wind, Droplets, Globe } from '@lucide/vue'

const props = defineProps({
  open: { type: Boolean, required: true },
  isHost: { type: Boolean, default: false },
  globalParams: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['close', 'save'])

const PARAM_META = {
  temperature: { label: '気温 (Temperature)', icon: Thermometer, color: 'text-amber-400', defaultUnit: '°C' },
  oxygen: { label: '酸素 (Oxygen)', icon: Wind, color: 'text-cyan-400', defaultUnit: '%' },
  oceans: { label: '海洋 (Oceans)', icon: Droplets, color: 'text-blue-400', defaultUnit: '枚' },
  venus: { label: '金星 (Venus Next)', icon: Globe, color: 'text-purple-400', defaultUnit: '%' },
}

const PRESETS = [
  { id: 'standard', name: '標準 (Standard / 基本ゲーム)' },
  { id: 'venus', name: 'ヴィーナス・ネクスト (Venus Next)' },
  { id: 'custom', name: 'カスタム (Custom)' },
]

const selectedPreset = ref('standard')
const form = ref({})

function getPresetParams(preset) {
  const base = {
    temperature: {
      id: 'temperature',
      name: 'Temperature',
      unit: '°C',
      current: -30,
      min: -30,
      max: 8,
      step: 2,
      enabled: true,
      required_end: true,
    },
    oxygen: {
      id: 'oxygen',
      name: 'Oxygen',
      unit: '%',
      current: 0,
      min: 0,
      max: 14,
      step: 1,
      enabled: true,
      required_end: true,
    },
    oceans: {
      id: 'oceans',
      name: 'Oceans',
      unit: '',
      current: 0,
      min: 0,
      max: 9,
      step: 1,
      enabled: true,
      required_end: true,
    },
    venus: {
      id: 'venus',
      name: 'Venus',
      unit: '%',
      current: 0,
      min: 0,
      max: 30,
      step: 2,
      enabled: false,
      required_end: false,
    },
  }

  if (preset === 'venus') {
    base.venus.enabled = true
  }
  return base
}

function initForm() {
  if (props.globalParams && Object.keys(props.globalParams).length > 0) {
    // Deep clone
    form.value = JSON.parse(JSON.stringify(props.globalParams))
    // Detect preset
    const isVenusEnabled = form.value.venus?.enabled
    if (isVenusEnabled) {
      selectedPreset.value = 'venus'
    } else {
      selectedPreset.value = 'standard'
    }
  } else {
    form.value = getPresetParams('standard')
    selectedPreset.value = 'standard'
  }
}

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    initForm()
  }
})

function applyPreset(presetId) {
  selectedPreset.value = presetId
  if (presetId !== 'custom') {
    const preset = getPresetParams(presetId)
    // Keep current values if valid
    for (const key of Object.keys(preset)) {
      if (form.value[key]) {
        preset[key].current = form.value[key].current
      }
    }
    form.value = preset
  }
}

const formError = computed(() => {
  for (const [id, p] of Object.entries(form.value)) {
    if (p.enabled) {
      if (p.step <= 0) return `${id}: 刻み (step) は 1 以上にしてください`
      if (p.min > p.max) return `${id}: 最小値が最大値を超えています`
      if (p.current < p.min || p.current > p.max) return `${id}: 現在値は最小値〜最大値の間に設定してください`
    }
  }
  return ''
})

function save() {
  if (formError.value) return
  emit('save', form.value)
  emit('close')
}
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-3 backdrop-blur-sm"
    @click.self="emit('close')"
  >
    <div class="relative max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-2xl border border-surface-border bg-surface-raised p-5 shadow-2xl">
      <div class="flex items-center justify-between border-b border-surface-border pb-3">
        <div class="flex items-center gap-2">
          <Sliders class="h-5 w-5 text-mars" />
          <h2 class="font-display text-lg font-bold text-ink">グローバルパラメータ設定</h2>
        </div>
        <button
          type="button"
          class="rounded-lg p-1.5 text-ink-muted hover:bg-surface hover:text-ink"
          @click="emit('close')"
        >
          <X class="h-5 w-5" />
        </button>
      </div>

      <div v-if="!isHost" class="my-3 rounded-lg bg-amber-950/40 p-2.5 text-xs text-amber-200">
        現在ホストのみがパラメータの範囲や有効状態を変更できます（設定の確認のみ可能）。
      </div>

      <!-- Presets -->
      <div v-if="isHost" class="mt-4">
        <label class="block text-xs font-semibold uppercase tracking-wider text-ink-muted">プリセット選択</label>
        <div class="mt-1.5 flex flex-wrap gap-2">
          <button
            v-for="preset in PRESETS"
            :key="preset.id"
            type="button"
            class="rounded-lg px-3 py-1.5 text-xs font-medium transition"
            :class="selectedPreset === preset.id
              ? 'bg-mars text-white shadow'
              : 'border border-surface-border bg-surface text-ink-muted hover:text-ink'"
            @click="applyPreset(preset.id)"
          >
            {{ preset.name }}
          </button>
        </div>
      </div>

      <!-- Parameters list -->
      <div class="mt-4 space-y-4">
        <div
          v-for="(meta, id) in PARAM_META"
          :key="id"
          class="rounded-xl border border-surface-border bg-surface/70 p-3.5 transition"
          :class="{ 'opacity-50': !form[id]?.enabled }"
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <component :is="meta.icon" class="h-5 w-5" :class="meta.color" />
              <span class="font-bold text-sm text-ink">{{ meta.label }}</span>
            </div>
            <label v-if="isHost" class="flex items-center gap-1.5 cursor-pointer text-xs text-ink-muted">
              <input
                v-model="form[id].enabled"
                type="checkbox"
                class="rounded border-surface-border text-mars focus:ring-mars"
                @change="selectedPreset = 'custom'"
              />
              <span>有効</span>
            </label>
            <span v-else class="text-xs text-ink-muted">
              {{ form[id]?.enabled ? '有効' : '無効' }}
            </span>
          </div>

          <div v-if="form[id]?.enabled" class="mt-3 grid grid-cols-2 gap-2.5 sm:grid-cols-4">
            <div>
              <label class="block text-[11px] text-ink-muted">最小値</label>
              <input
                v-model.number="form[id].min"
                type="number"
                :disabled="!isHost"
                class="mt-1 w-full rounded-md border border-surface-border bg-surface-raised px-2 py-1 text-xs text-ink focus:border-mars focus:outline-none disabled:opacity-75"
                @input="selectedPreset = 'custom'"
              />
            </div>
            <div>
              <label class="block text-[11px] text-ink-muted">最大値</label>
              <input
                v-model.number="form[id].max"
                type="number"
                :disabled="!isHost"
                class="mt-1 w-full rounded-md border border-surface-border bg-surface-raised px-2 py-1 text-xs text-ink focus:border-mars focus:outline-none disabled:opacity-75"
                @input="selectedPreset = 'custom'"
              />
            </div>
            <div>
              <label class="block text-[11px] text-ink-muted">1回の刻み</label>
              <input
                v-model.number="form[id].step"
                type="number"
                min="1"
                :disabled="!isHost"
                class="mt-1 w-full rounded-md border border-surface-border bg-surface-raised px-2 py-1 text-xs text-ink focus:border-mars focus:outline-none disabled:opacity-75"
                @input="selectedPreset = 'custom'"
              />
            </div>
            <div>
              <label class="block text-[11px] text-ink-muted">現在値</label>
              <input
                v-model.number="form[id].current"
                type="number"
                :disabled="!isHost"
                class="mt-1 w-full rounded-md border border-surface-border bg-surface-raised px-2 py-1 text-xs text-ink focus:border-mars focus:outline-none disabled:opacity-75"
                @input="selectedPreset = 'custom'"
              />
            </div>
          </div>

          <div v-if="form[id]?.enabled" class="mt-2.5 flex items-center justify-between text-xs">
            <label v-if="isHost" class="flex items-center gap-1.5 cursor-pointer text-ink-muted hover:text-ink">
              <input
                v-model="form[id].required_end"
                type="checkbox"
                class="rounded border-surface-border text-mars focus:ring-mars"
                @change="selectedPreset = 'custom'"
              />
              <span>ゲーム終了条件（最終世代判定）に含める</span>
            </label>
            <span v-else class="text-ink-muted">
              終了条件: {{ form[id]?.required_end ? '含む' : '含まない' }}
            </span>
          </div>
        </div>
      </div>

      <p v-if="formError" class="mt-3 rounded-lg bg-red-950/60 px-3 py-2 text-xs text-red-300">
        {{ formError }}
      </p>

      <div class="mt-5 flex justify-end gap-2 border-t border-surface-border pt-4">
        <button
          type="button"
          class="rounded-lg border border-surface-border px-4 py-2 text-xs font-medium text-ink-muted hover:bg-surface hover:text-ink"
          @click="emit('close')"
        >
          閉じる
        </button>
        <button
          v-if="isHost"
          type="button"
          :disabled="Boolean(formError)"
          class="inline-flex items-center gap-1.5 rounded-lg bg-mars px-4 py-2 text-xs font-bold text-white shadow hover:brightness-110 disabled:opacity-50"
          @click="save"
        >
          <Check class="h-4 w-4" />
          保存して適用
        </button>
      </div>
    </div>
  </div>
</template>
