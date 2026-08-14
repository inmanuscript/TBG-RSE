<script setup>
import { computed, ref } from 'vue'
import { Thermometer, Wind, Droplets, Globe, Sliders, Plus, Minus, Check, X, Sparkles } from '@lucide/vue'

const props = defineProps({
  globalParams: { type: Object, default: () => ({}) },
  isHost: { type: Boolean, default: false },
})

const emit = defineEmits(['update-param', 'open-config'])

const PARAM_META = {
  temperature: {
    label: '気温',
    icon: Thermometer,
    textColor: 'text-amber-400',
    barColor: 'bg-gradient-to-r from-blue-500 via-amber-500 to-red-500',
    bgColor: 'bg-amber-500/10 border-amber-500/30',
  },
  oxygen: {
    label: '酸素',
    icon: Wind,
    textColor: 'text-cyan-400',
    barColor: 'bg-cyan-500',
    bgColor: 'bg-cyan-500/10 border-cyan-500/30',
  },
  oceans: {
    label: '海洋',
    icon: Droplets,
    textColor: 'text-blue-400',
    barColor: 'bg-blue-500',
    bgColor: 'bg-blue-500/10 border-blue-500/30',
  },
  venus: {
    label: '金星',
    icon: Globe,
    textColor: 'text-purple-400',
    barColor: 'bg-purple-500',
    bgColor: 'bg-purple-500/10 border-purple-500/30',
  },
}

const activeParams = computed(() => {
  if (!props.globalParams) return []
  return Object.values(props.globalParams).filter((p) => p.enabled)
})

const isAllMaxed = computed(() => {
  const req = activeParams.value.filter((p) => p.required_end)
  return req.length > 0 && req.every((p) => p.current >= p.max)
})

function getProgressPercent(param) {
  if (!param || param.max <= param.min) return 0
  const pct = Math.round(((param.current - param.min) / (param.max - param.min)) * 100)
  return Math.max(0, Math.min(100, pct))
}

// Quick edit modal state
const activeEditId = ref(null)
const deltaStep = ref(1)
const grantTR = ref(true)

const selectedParam = computed(() => {
  if (!activeEditId.value || !props.globalParams) return null
  return props.globalParams[activeEditId.value] || null
})

function openQuickEdit(paramId) {
  activeEditId.value = paramId
  deltaStep.value = 1
  grantTR.value = true
}

function closeQuickEdit() {
  activeEditId.value = null
}

function submitUpdate(steps) {
  if (!selectedParam.value) return
  emit('update-param', {
    paramId: selectedParam.value.id,
    deltaSteps: steps,
    grantTR: grantTR.value,
  })
  closeQuickEdit()
}
</script>

<template>
  <div class="rounded-2xl border border-surface-border bg-surface-raised/80 p-3 shadow backdrop-blur">
    <div class="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border/60 pb-2">
      <div class="flex items-center gap-2">
        <span class="font-display text-xs font-bold uppercase tracking-wider text-ink-muted">グローバルパラメータ</span>
        <span
          v-if="isAllMaxed"
          class="inline-flex items-center gap-1 rounded-full bg-emerald-950/80 border border-emerald-500/40 px-2 py-0.5 text-[11px] font-bold text-emerald-300 animate-pulse"
        >
          🏁 最終世代 (パラメータ達成)
        </span>
      </div>
      <button
        type="button"
        class="inline-flex items-center gap-1 rounded-lg border border-surface-border bg-surface px-2 py-1 text-xs text-ink-muted hover:border-mars hover:text-ink transition"
        title="グローバルパラメータ設定"
        @click="emit('open-config')"
      >
        <Sliders class="h-3.5 w-3.5 text-mars" />
        <span>設定</span>
      </button>
    </div>

    <!-- Parameter cards grid -->
    <div class="mt-2.5 grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
      <div
        v-for="param in activeParams"
        :key="param.id"
        class="group relative flex cursor-pointer flex-col justify-between rounded-xl border p-2.5 transition hover:brightness-110 active:scale-[0.98]"
        :class="PARAM_META[param.id]?.bgColor || 'border-surface-border bg-surface'"
        @click="openQuickEdit(param.id)"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1.5">
            <component
              :is="PARAM_META[param.id]?.icon || Globe"
              class="h-4 w-4"
              :class="PARAM_META[param.id]?.textColor || 'text-ink'"
            />
            <span class="text-xs font-semibold text-ink">
              {{ PARAM_META[param.id]?.label || param.name }}
            </span>
          </div>
          <span
            v-if="param.current >= param.max"
            class="rounded bg-emerald-500/20 px-1 text-[10px] font-bold text-emerald-400"
          >
            MAX
          </span>
        </div>

        <div class="my-1.5 flex items-baseline justify-between">
          <span class="font-display text-base font-bold text-ink">
            {{ param.current }}{{ param.unit }}
          </span>
          <span class="text-[11px] text-ink-muted">
            / {{ param.max }}{{ param.unit }}
          </span>
        </div>

        <!-- Progress bar -->
        <div class="h-1.5 w-full overflow-hidden rounded-full bg-surface-border/50">
          <div
            class="h-full rounded-full transition-all duration-300"
            :class="PARAM_META[param.id]?.barColor || 'bg-mars'"
            :style="{ width: `${getProgressPercent(param)}%` }"
          />
        </div>
      </div>
    </div>

    <!-- Quick edit modal / popover -->
    <div
      v-if="selectedParam"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-3 backdrop-blur-sm"
      @click.self="closeQuickEdit"
    >
      <div class="relative w-full max-w-sm rounded-2xl border border-surface-border bg-surface-raised p-5 shadow-2xl">
        <div class="flex items-center justify-between border-b border-surface-border pb-3">
          <div class="flex items-center gap-2">
            <component
              :is="PARAM_META[selectedParam.id]?.icon || Globe"
              class="h-5 w-5"
              :class="PARAM_META[selectedParam.id]?.textColor || 'text-ink'"
            />
            <h3 class="font-display font-bold text-ink">
              {{ PARAM_META[selectedParam.id]?.label || selectedParam.name }} 操作
            </h3>
          </div>
          <button
            type="button"
            class="rounded-lg p-1 text-ink-muted hover:bg-surface hover:text-ink"
            @click="closeQuickEdit"
          >
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="mt-4 text-center">
          <div class="text-xs text-ink-muted">現在の値</div>
          <div class="font-display text-3xl font-bold text-ink">
            {{ selectedParam.current }}{{ selectedParam.unit }}
          </div>
          <div class="mt-1 text-xs text-ink-muted">
            範囲: {{ selectedParam.min }}{{ selectedParam.unit }} 〜 {{ selectedParam.max }}{{ selectedParam.unit }} (1刻み: +{{ selectedParam.step }}{{ selectedParam.unit }})
          </div>
          <div v-if="selectedParam.current >= selectedParam.max" class="mt-2 text-xs font-semibold text-amber-400">
            ⚠️ 既に最大値に達しています（上昇時のTRは得られません）
          </div>
        </div>

        <div class="mt-4 rounded-xl border border-surface-border bg-surface/50 p-3">
          <label class="flex items-center justify-between cursor-pointer text-xs text-ink">
            <span>TR連動 (上限未満ならTRも加算)</span>
            <input
              v-model="grantTR"
              type="checkbox"
              class="rounded border-surface-border text-mars focus:ring-mars"
            />
          </label>
        </div>

        <div class="mt-5 grid grid-cols-2 gap-2">
          <button
            type="button"
            :disabled="selectedParam.current <= selectedParam.min"
            class="inline-flex items-center justify-center gap-1 rounded-xl border border-surface-border bg-surface py-2.5 text-xs font-bold text-ink hover:border-red-500/50 hover:bg-red-500/10 active:scale-95 disabled:opacity-40"
            @click="submitUpdate(-1)"
          >
            <Minus class="h-4 w-4" />
            −{{ selectedParam.step }}{{ selectedParam.unit }}
          </button>
          <button
            type="button"
            :disabled="selectedParam.current >= selectedParam.max"
            class="inline-flex items-center justify-center gap-1 rounded-xl border border-surface-border bg-surface py-2.5 text-xs font-bold text-ink hover:border-emerald-500/50 hover:bg-emerald-500/10 active:scale-95 disabled:opacity-40"
            @click="submitUpdate(1)"
          >
            <Plus class="h-4 w-4" />
            +{{ selectedParam.step }}{{ selectedParam.unit }}
          </button>
        </div>

        <button
          type="button"
          class="mt-3 w-full rounded-lg border border-surface-border py-1.5 text-xs text-ink-muted hover:bg-surface hover:text-ink"
          @click="closeQuickEdit"
        >
          キャンセル
        </button>
      </div>
    </div>
  </div>
</template>
