<script setup>
import { computed, ref, watch, onMounted } from 'vue'
import { Copy, Check, Leaf, Flame, ScrollText, Wifi, WifiOff, SkipForward, Flag, Trophy, Zap, ChevronDown, ChevronUp, X, AlertCircle } from '@lucide/vue'
import ResourceCard from './ResourceCard.vue'
import OpponentCard from './OpponentCard.vue'
import VPHelper from './VPHelper.vue'
import RepeatPressButton from './RepeatPressButton.vue'
import CountKeypad from './CountKeypad.vue'
import GlobalParamsBar from './GlobalParamsBar.vue'
import GlobalParamConfigModal from './GlobalParamConfigModal.vue'

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
  isMyTurn: { type: Boolean, required: true },
  activePlayer: { type: Object, default: null },
  playerId: { type: String, required: true },
  isHost: { type: Boolean, default: false },
  error: { type: String, default: '' },
})

const emit = defineEmits([
  'update', 'ready', 'shortcut', 'project', 'buy-cards',
  'end-turn', 'pass', 'claim-action', 'tag', 'score', 'end-game', 'activity', 'leave',
  'global-param', 'configure-global-params', 'clear-error',
])

const cardBuy = ref(0)
const cardBuyKeypadOpen = ref(false)
const sellCount = ref(1)
const tagsExpanded = ref(false)
const configModalOpen = ref(false)

onMounted(() => {
  tagsExpanded.value = window.matchMedia('(min-width: 640px)').matches
})

// Generation 1 deals a larger starting hand (up to 10); later generations cap at 4.
// Mirrors game.MaxCardsBuyForGeneration on the server.
const maxCardsBuy = computed(() => (props.state.generation <= 1 ? 10 : 4))
const cardBuyOptions = computed(() => Array.from({ length: maxCardsBuy.value + 1 }, (_, i) => i))
// A pill per value reads fine up to ~5 options; beyond that a keypad stays compact.
const useCardBuyKeypad = computed(() => maxCardsBuy.value > 5)

watch(maxCardsBuy, (max) => {
  if (cardBuy.value > max) cardBuy.value = max
})

function submitCardBuyKeypad(value) {
  cardBuy.value = value
  cardBuyKeypadOpen.value = false
}

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
// End Turn and Pass are mutually exclusive server-side (production.go EndTurn/Pass):
// End Turn requires an action already taken, Pass requires none yet. Only offer
// the one that would actually succeed instead of showing both and erroring.
const hasActedThisTurn = computed(() => (props.state.actions_this_turn || 0) >= 1)

const roomCodeCopied = ref(false)
let roomCodeCopyTimer = null

// navigator.clipboard is only available in secure contexts (HTTPS/localhost) —
// this app is commonly self-hosted and opened over plain HTTP on a LAN, where
// that API is simply undefined and writeText() silently no-ops. Fall back to
// the legacy execCommand copy, which still works over HTTP.
function legacyCopy(text) {
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.focus()
  ta.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  document.body.removeChild(ta)
  return ok
}

async function copyCode() {
  let ok = false
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(props.roomCode)
      ok = true
    }
  } catch {
    ok = false
  }
  if (!ok) ok = legacyCopy(props.roomCode)

  if (ok) {
    roomCodeCopied.value = true
    clearTimeout(roomCodeCopyTimer)
    roomCodeCopyTimer = setTimeout(() => {
      roomCodeCopied.value = false
    }, 1500)
  }
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
    <p v-if="!connected" class="mb-4 rounded-lg bg-amber-950/50 px-3 py-2 text-sm text-amber-200">
      サーバーに再接続中です。オンラインになるまで操作できません。
    </p>

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
              :class="{ 'border-emerald-600 text-emerald-300': roomCodeCopied }"
              @click="copyCode"
            >
              <Check v-if="roomCodeCopied" class="h-3 w-3" />
              <Copy v-else class="h-3 w-3" />
              {{ roomCodeCopied ? 'コピーしました' : roomCode }}
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
            v-if="state.phase !== 'ENDED'"
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
              v-if="isMyTurn && hasActedThisTurn"
              type="button"
              class="inline-flex items-center gap-1 rounded-lg bg-surface-border px-3 py-2 text-sm hover:bg-mars"
              @click="$emit('end-turn')"
            >
              <SkipForward class="h-4 w-4" />
              ターン終了
            </button>
            <button
              v-if="isMyTurn && !hasActedThisTurn"
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
    </header>

    <!-- Global Parameters Bar -->
    <div class="mb-4">
      <GlobalParamsBar
        :global-params="state.global_params"
        :is-host="isHost"
        @update-param="(p) => $emit('global-param', p)"
        @open-config="configModalOpen = true"
      />
    </div>

    <!-- Global Parameters Config Modal -->
    <GlobalParamConfigModal
      :open="configModalOpen"
      :is-host="isHost"
      :global-params="state.global_params"
      @save="(p) => $emit('configure-global-params', p)"
      @close="configModalOpen = false"
    />

    <!-- Research -->
    <section
      v-if="state.phase === 'RESEARCH'"
      class="mb-4 rounded-2xl border border-cyan-900/40 bg-cyan-950/20 p-4"
    >
      <h2 class="font-display text-sm tracking-wide text-cyan-200">研究フェイズ — カード購入</h2>
      <p class="mt-1 text-sm text-ink-muted">
        {{ useCardBuyKeypad ? '初期手札10枚想定。' : '4枚ドロー想定。' }}3 MC / 枚。
      </p>
      <div v-if="!me.research_done" class="mt-3 flex flex-wrap items-center gap-3">
        <button
          v-if="useCardBuyKeypad"
          type="button"
          class="rounded-lg border border-surface-border bg-surface px-4 py-1.5 font-display text-xl font-bold tabular-nums hover:border-cyan-500"
          title="タップして購入枚数を入力"
          @click="cardBuyKeypadOpen = true"
        >
          {{ cardBuy }}
        </button>
        <div v-else class="flex gap-1.5" role="radiogroup" aria-label="購入枚数">
          <button
            v-for="n in cardBuyOptions"
            :key="n"
            type="button"
            role="radio"
            :aria-checked="cardBuy === n"
            class="h-9 w-9 rounded-full text-sm font-semibold transition"
            :class="cardBuy === n ? 'bg-cyan-600 text-white' : 'bg-surface text-ink-muted hover:bg-surface-border hover:text-ink'"
            @click="cardBuy = n"
          >
            {{ n }}
          </button>
        </div>
        <span class="text-sm text-ink-muted">= {{ cardBuy * 3 }} MC</span>
        <button
          type="button"
          class="rounded-xl bg-cyan-700 px-4 py-2 text-sm font-semibold text-white hover:bg-cyan-600"
          @click="$emit('buy-cards', cardBuy)"
        >
          購入確定
        </button>

        <CountKeypad
          v-if="cardBuyKeypadOpen"
          title="購入枚数"
          :min="0"
          :max="maxCardsBuy"
          @submit="submitCardBuyKeypad"
          @cancel="cardBuyKeypadOpen = false"
        />
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
                v-for="d in [-1, 1]"
                :key="'tr' + d"
                class="rounded-lg bg-surface px-3 py-2 text-sm font-semibold hover:bg-surface-border"
                :disabled="!connected"
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
                  :disabled="!connected"
                  @press="$emit('tag', { tag, delta: -1 })"
                >
                  −
                </RepeatPressButton>
                <RepeatPressButton
                  class="rounded bg-surface-border px-1.5 text-xs"
                  :disabled="!connected"
                  @press="$emit('tag', { tag, delta: 1 })"
                >
                  +
                </RepeatPressButton>
              </div>
            </div>
          </div>
        </div>

        <div class="resource-grid grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <ResourceCard
            v-for="key in resourceOrder"
            :key="key"
            :resource-key="key"
            :meta="resourceMeta[key]"
            :stock="me.resources?.[key]?.stock ?? 0"
            :production="me.resources?.[key]?.production ?? 0"
            :interactive="connected"
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

<style scoped>
/* スマホ横画面(iPhone SEなど幅568px程度の端末を含む)では sm: の
   640px幅ブレークポイントに届かず1列(6行)のままになる。基本6資源を
   横画面時は必ず2x3で並べたいので、幅に関わらずorientationで2列を強制する。
   1024px以上(タブレット横画面〜PC)はxl:grid-cols-3など既存の幅ベースの
   レイアウトを尊重し、対象外とする。 */
@media (orientation: landscape) and (max-width: 1023px) {
  .resource-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
