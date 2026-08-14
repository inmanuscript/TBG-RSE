<script setup>
import { computed, ref, onMounted } from 'vue'
import { Copy, Leaf, Flame, ScrollText, Wifi, WifiOff, SkipForward, Flag, Trophy, Zap, ChevronDown, ChevronUp } from '@lucide/vue'
import ResourceCard from './ResourceCard.vue'
import OpponentCard from './OpponentCard.vue'
import VPHelper from './VPHelper.vue'
import RepeatPressButton from './RepeatPressButton.vue'

const props = defineProps({
  state: { type: Object, required: true },
  me: { type: Object, required: true },
  opponents: { type: Array, required: true },
  orderedPlayers: { type: Array, required: true },
  roomCode: { type: String, required: true },
  connected: { type: Boolean, required: true },
  resourceOrder: { type: Array, required: true },
  resourceMeta: { type: Object, required: true },
  tags: { type: Array, required: true },
  projects: { type: Array, required: true },
  scoreFields: { type: Array, required: true },
  lastHighlight: { type: Object, required: true },
  isHost: { type: Boolean, required: true },
  isMyTurn: { type: Boolean, required: true },
  activePlayer: { type: Object, default: null },
  playerId: { type: String, required: true },
  error: { type: String, default: '' },
})

const emit = defineEmits([
  'update', 'ready', 'shortcut', 'project', 'buy-cards',
  'end-turn', 'pass', 'claim-action', 'tag', 'score', 'end-game', 'activity', 'leave',
])

const cardBuy = ref(0)
const sellCount = ref(1)
const tagsExpanded = ref(false)

onMounted(() => {
  tagsExpanded.value = window.matchMedia('(min-width: 640px)').matches
})

const tagSummary = computed(() =>
  props.tags
    .filter((t) => (props.me.tags?.[t] || 0) > 0)
    .map((t) => `${t} ${props.me.tags[t]}`),
)

const phaseLabel = computed(() => {
  switch (props.state.phase) {
    case 'RESEARCH': return '研究フェイズ'
    case 'ACTION': return 'アクション'
    case 'PRODUCTION_WAIT': return '産出待機'
    case 'ENDED': return '終了 / VP'
    default: return props.state.phase
  }
})

const actionsLeft = computed(() => Math.max(0, 2 - (props.state.actions_this_turn || 0)))

function copyCode() {
  navigator.clipboard?.writeText(props.roomCode)
}

function isHighlighted(playerId) {
  const ts = props.lastHighlight[playerId]
  return ts && Date.now() - ts < 3500
}

function onProject(p) {
  if (p.needsCards) {
    emit('project', { kind: p.kind, cardsSold: sellCount.value })
  } else {
    emit('project', { kind: p.kind, cardsSold: 0 })
  }
}
</script>

<template>
  <div class="mx-auto min-h-screen max-w-7xl px-3 pb-10 pt-3 sm:px-5">
    <header class="mb-4 rounded-2xl border border-surface-border bg-surface-raised/80 p-4 backdrop-blur">
      <div class="flex flex-wrap items-center gap-3 justify-between">
        <div>
          <p class="font-display text-[11px] tracking-[0.28em] text-mars-glow">TBG-RSE</p>
          <h1 class="font-display text-2xl font-bold sm:text-3xl">
            Generation
            <span class="text-mars">{{ state.generation }}</span>
          </h1>
          <p class="mt-1 flex flex-wrap items-center gap-2 text-xs text-ink-muted">
            <button
              type="button"
              class="inline-flex items-center gap-1 rounded-md border border-surface-border bg-surface px-2 py-1 font-display tracking-widest text-ink hover:border-mars"
              @click="copyCode"
            >
              <Copy class="h-3 w-3" />
              {{ roomCode }}
            </button>
            <span>{{ phaseLabel }}</span>
            <span class="inline-flex items-center gap-1">
              <Wifi v-if="connected" class="h-3.5 w-3.5 text-emerald-400" />
              <WifiOff v-else class="h-3.5 w-3.5 text-red-400" />
              {{ connected ? 'online' : 'reconnecting…' }}
            </span>
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <button
            type="button"
            class="inline-flex items-center gap-2 rounded-xl border border-surface-border bg-surface px-3 py-2 text-sm hover:border-mars"
            @click="$emit('activity')"
          >
            <ScrollText class="h-4 w-4" />
            ログ
          </button>
          <button
            v-if="isHost && state.phase !== 'ENDED'"
            type="button"
            class="inline-flex items-center gap-2 rounded-xl border border-amber-800/50 bg-amber-950/30 px-3 py-2 text-sm text-amber-200 hover:border-amber-500"
            @click="$emit('end-game')"
          >
            <Trophy class="h-4 w-4" />
            終了
          </button>
          <button type="button" class="text-xs text-ink-muted underline-offset-2 hover:underline" @click="$emit('leave')">
            退出
          </button>
        </div>
      </div>

      <!-- Turn strip -->
      <div v-if="state.phase === 'ACTION' || state.phase === 'PRODUCTION_WAIT'" class="mt-4 rounded-xl bg-surface p-3">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="text-sm">
            <template v-if="state.phase === 'PRODUCTION_WAIT'">
              全員パス済み — 産出を実行してください
            </template>
            <template v-else>
              手番:
              <span class="font-semibold" :style="{ color: activePlayer?.color }">
                {{ activePlayer?.name || '—' }}
              </span>
              <span class="ml-2 text-ink-muted">アクション残 {{ actionsLeft }}/2</span>
            </template>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              v-if="isMyTurn"
              type="button"
              class="inline-flex items-center gap-2 rounded-xl bg-mars px-4 py-2.5 text-sm font-semibold text-white hover:bg-mars-glow animate-pulseReady"
              @click="$emit('claim-action')"
            >
              <Zap class="h-4 w-4" />
              アクション実施
            </button>
            <button
              v-if="isMyTurn"
              type="button"
              class="inline-flex items-center gap-1 rounded-lg bg-surface-border px-3 py-2 text-sm hover:bg-mars"
              @click="$emit('end-turn')"
            >
              <SkipForward class="h-4 w-4" />
              ターン終了
            </button>
            <button
              v-if="isMyTurn"
              type="button"
              class="inline-flex items-center gap-1 rounded-lg border border-red-800/50 bg-red-950/40 px-3 py-2 text-sm text-red-200"
              @click="$emit('pass')"
            >
              <Flag class="h-4 w-4" />
              パス（世代脱落）
            </button>
            <button
              v-if="state.phase === 'PRODUCTION_WAIT'"
              type="button"
              class="rounded-xl bg-mars px-5 py-2.5 font-semibold text-white hover:bg-mars-glow"
              @click="$emit('ready')"
            >
              産出実行
            </button>
          </div>
        </div>
        <div class="mt-3 flex flex-wrap gap-2">
          <span
            v-for="p in orderedPlayers"
            :key="p.id"
            class="rounded-full border px-2.5 py-1 text-xs"
            :class="{
              'border-mars text-mars-glow': state.active_player_id === p.id,
              'border-surface-border text-ink-muted line-through opacity-50': p.passed,
              'border-surface-border text-ink': state.active_player_id !== p.id && !p.passed,
            }"
          >
            <span class="mr-1 opacity-70">{{ p.seat }}.</span>{{ p.name }}
          </span>
        </div>
      </div>

      <p v-if="error" class="mt-3 rounded-lg bg-red-950/50 px-3 py-2 text-sm text-red-300">{{ error }}</p>
    </header>

    <!-- Research -->
    <section
      v-if="state.phase === 'RESEARCH'"
      class="mb-4 rounded-2xl border border-cyan-900/40 bg-cyan-950/20 p-4"
    >
      <h2 class="font-display text-sm tracking-wide text-cyan-200">研究フェイズ — カード購入</h2>
      <p class="mt-1 text-sm text-ink-muted">4枚ドロー想定。購入は 3 MC / 枚（0〜4）。</p>
      <div v-if="!me.research_done" class="mt-3 flex flex-wrap items-center gap-3">
        <input
          v-model.number="cardBuy"
          type="number"
          min="0"
          max="4"
          class="w-20 rounded-lg border border-surface-border bg-surface px-3 py-2"
        />
        <span class="text-sm text-ink-muted">枚 = {{ (cardBuy || 0) * 3 }} MC</span>
        <button
          type="button"
          class="rounded-xl bg-cyan-700 px-4 py-2 text-sm font-semibold text-white hover:bg-cyan-600"
          @click="$emit('buy-cards', cardBuy || 0)"
        >
          購入確定
        </button>
      </div>
      <p v-else class="mt-3 text-sm text-emerald-300">購入済み — 他プレイヤー待ち</p>
      <ul class="mt-3 flex flex-wrap gap-2 text-xs">
        <li
          v-for="p in orderedPlayers"
          :key="p.id"
          class="rounded-md border border-surface-border px-2 py-1"
          :class="p.research_done ? 'text-emerald-300' : 'text-ink-muted'"
        >
          {{ p.name }}: {{ p.research_done ? '済' : '未' }}
        </li>
      </ul>
    </section>

    <!-- VP helper -->
    <VPHelper
      v-if="state.phase === 'ENDED'"
      :players="orderedPlayers"
      :player-id="playerId"
      :score-fields="scoreFields"
      @score="(p) => $emit('score', p)"
    />

    <div v-if="state.phase !== 'ENDED'" class="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
      <section>
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h2 class="font-display text-sm tracking-wide text-ink-muted">My Board</h2>
          <div class="flex items-center gap-2">
            <span class="h-3 w-3 rounded-full" :style="{ backgroundColor: me.color }" />
            <span class="font-semibold">{{ me.name }}</span>
            <span v-if="me.passed" class="text-xs text-red-300">PASSED</span>
          </div>
        </div>

        <div class="mb-4 rounded-2xl border border-surface-border bg-surface-raised/80 p-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-[10px] uppercase tracking-wider text-ink-muted">Terraforming Rating</p>
              <p class="font-display text-4xl font-bold tabular-nums">{{ me.tr }}</p>
            </div>
            <div class="flex flex-wrap gap-1.5">
              <RepeatPressButton
                v-for="d in [-10, -1, 1, 10]"
                :key="'tr' + d"
                class="rounded-lg bg-surface px-3 py-2 text-sm font-semibold hover:bg-surface-border"
                @press="$emit('update', { target: 'tr', delta: d })"
              >
                {{ d > 0 ? '+' : '' }}{{ d }}
              </RepeatPressButton>
            </div>
          </div>

          <div class="mt-4">
            <p class="mb-2 text-[10px] uppercase tracking-wider text-ink-muted">Conversions / Standard Projects</p>
            <div class="flex flex-wrap gap-2">
              <button
                type="button"
                class="inline-flex items-center gap-2 rounded-xl border border-emerald-800/60 bg-emerald-950/40 px-3 py-2 text-sm text-emerald-200 disabled:opacity-40"
                :disabled="!isMyTurn"
                @click="$emit('shortcut', 'greenery')"
              >
                <Leaf class="h-4 w-4" />
                植物8 → 緑化
              </button>
              <button
                type="button"
                class="inline-flex items-center gap-2 rounded-xl border border-orange-800/60 bg-orange-950/40 px-3 py-2 text-sm text-orange-200 disabled:opacity-40"
                :disabled="!isMyTurn"
                @click="$emit('shortcut', 'temperature')"
              >
                <Flame class="h-4 w-4" />
                発熱8 → 温度
              </button>
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-2">
              <label class="text-xs text-ink-muted">
                売却枚数
                <input v-model.number="sellCount" type="number" min="1" max="20" class="ml-1 w-14 rounded border border-surface-border bg-surface px-2 py-1" />
              </label>
              <button
                v-for="p in projects"
                :key="p.kind"
                type="button"
                class="rounded-lg border border-surface-border bg-surface px-2.5 py-1.5 text-xs hover:border-mars disabled:opacity-40"
                :disabled="!isMyTurn"
                :title="p.cost"
                @click="onProject(p)"
              >
                {{ p.label }}
                <span class="text-ink-muted">({{ p.cost }})</span>
              </button>
            </div>
            <p v-if="state.phase === 'ACTION' && !isMyTurn" class="mt-2 text-xs text-ink-muted">
              標準プロジェクト／変換は自分の手番のみ（資源の手動調整はいつでも可）
            </p>
          </div>
        </div>

        <div class="mb-4 rounded-2xl border border-surface-border bg-surface-raised/70 p-4">
          <button
            type="button"
            class="flex w-full items-start justify-between gap-3 text-left"
            :aria-expanded="tagsExpanded"
            @click="tagsExpanded = !tagsExpanded"
          >
            <div class="min-w-0">
              <h3 class="font-display text-xs tracking-wide text-ink-muted">Tags</h3>
              <p v-if="!tagsExpanded && tagSummary.length" class="mt-1 truncate text-xs text-ink">
                {{ tagSummary.join(' · ') }}
              </p>
              <p v-else-if="!tagsExpanded" class="mt-1 text-xs text-ink-muted">タップで展開</p>
            </div>
            <ChevronDown v-if="!tagsExpanded" class="mt-0.5 h-4 w-4 shrink-0 text-ink-muted" />
            <ChevronUp v-else class="mt-0.5 h-4 w-4 shrink-0 text-ink-muted" />
          </button>
          <div v-show="tagsExpanded" class="mt-3 grid grid-cols-4 gap-1.5 sm:grid-cols-4 sm:gap-2 md:grid-cols-6">
            <div v-for="tag in tags" :key="tag" class="rounded-lg bg-surface px-1.5 py-1.5 text-center sm:px-2 sm:py-2">
              <p class="truncate text-[9px] text-ink-muted sm:text-[10px]">{{ tag }}</p>
              <p class="font-display text-base tabular-nums sm:text-lg">{{ me.tags?.[tag] || 0 }}</p>
              <div class="mt-0.5 flex justify-center gap-1 sm:mt-1">
                <RepeatPressButton
                  class="rounded bg-surface-border px-1.5 text-xs"
                  @press="$emit('tag', { tag, delta: -1 })"
                >
                  −
                </RepeatPressButton>
                <RepeatPressButton
                  class="rounded bg-surface-border px-1.5 text-xs"
                  @press="$emit('tag', { tag, delta: 1 })"
                >
                  +
                </RepeatPressButton>
              </div>
            </div>
          </div>
        </div>

        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <ResourceCard
            v-for="key in resourceOrder"
            :key="key"
            :resource-key="key"
            :meta="resourceMeta[key]"
            :stock="me.resources?.[key]?.stock ?? 0"
            :production="me.resources?.[key]?.production ?? 0"
            @change="(p) => $emit('update', p)"
          />
        </div>
      </section>

      <section>
        <h2 class="mb-3 font-display text-sm tracking-wide text-ink-muted">Opponents</h2>
        <div class="space-y-3">
          <OpponentCard
            v-for="op in opponents"
            :key="op.id"
            :player="op"
            :resource-order="resourceOrder"
            :resource-meta="resourceMeta"
            :highlighted="isHighlighted(op.id)"
            :active="state.active_player_id === op.id"
          />
          <p
            v-if="!opponents.length"
            class="rounded-2xl border border-dashed border-surface-border px-4 py-10 text-center text-sm text-ink-muted"
          >
            他プレイヤーの参加を待っています…
            <br />
            コード <span class="font-display tracking-widest text-ink">{{ roomCode }}</span> を共有
          </p>
        </div>
      </section>
    </div>
  </div>
</template>
