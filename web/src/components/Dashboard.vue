<script setup>
import { computed, ref, watch, onMounted } from 'vue'
import { Copy, Check, Leaf, Flame, ScrollText, Wifi, WifiOff, SkipForward, Flag, Trophy, Zap, ChevronDown, ChevronUp, ChevronRight, Hammer, Tags, X, AlertCircle } from '@lucide/vue'
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
  'global-param', 'configure-global-params', 'clear-error', 'skip-player',
])

const cardBuy = ref(0)
const cardBuyKeypadOpen = ref(false)
const sellCount = ref(1)
const tagsExpanded = ref(false)
const projectsExpanded = ref(false)
const configModalOpen = ref(false)
// モバイル固定レイアウト用のボトムシート — null | 'projects' | 'tags' | 'vp' | 'opponent'
const mobileSheet = ref(null)
const mobileSheetOpponent = ref(null)
// 研究フェーズパネルは「未購入なら展開/購入済みなら折りたたみ」を初期値とし、
// 自分の購入完了・新しい研究フェーズ突入のたびに追従させる(#19: 見逃し対策)。
const researchExpanded = ref(!props.me.research_done)

onMounted(() => {
  const isDesktop = window.matchMedia('(min-width: 640px)').matches
  tagsExpanded.value = isDesktop
  projectsExpanded.value = isDesktop
})

watch(() => props.me.research_done, (done) => {
  if (done) researchExpanded.value = false
})
watch(() => props.state.phase, (phase) => {
  if (phase === 'RESEARCH') researchExpanded.value = !props.me.research_done
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

function openMobileSheet(name) {
  mobileSheetOpponent.value = null
  mobileSheet.value = name
}

function openOpponentSheet(op) {
  mobileSheetOpponent.value = op
  mobileSheet.value = 'opponent'
}

function closeMobileSheet() {
  mobileSheet.value = null
  mobileSheetOpponent.value = null
}

const mobileSheetTitle = computed(() => {
  switch (mobileSheet.value) {
    case 'projects': return '標準プロジェクト'
    case 'tags': return 'タグ'
    case 'vp': return 'VP'
    case 'opponent': return mobileSheetOpponent.value?.name || ''
    default: return ''
  }
})
</script>

<template>
  <!-- デスクトップ/タブレット: 従来どおりの縦スクロールレイアウト -->
  <div class="mx-auto hidden min-h-screen max-w-7xl px-3 pb-10 pt-3 sm:block sm:px-5">
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

    <!-- VP helper -->
    <VPHelper
      v-if="state.phase === 'ENDED'"
      :players="orderedPlayers"
      :player-id="playerId"
      :score-fields="scoreFields"
      @score="(p) => $emit('score', p)"
    />

    <template v-if="state.phase !== 'ENDED'">
      <!-- 2. タグカウントヘルパ -->
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

      <!-- 3. アクション管理 — スクロールして資源パネル等を見ていても見失わないようsticky表示 -->
      <div
        v-if="state.phase === 'ACTION' || state.phase === 'PRODUCTION_WAIT'"
        class="sticky top-2 z-20 mb-4 rounded-2xl border border-mars/40 bg-surface-raised/95 p-3 shadow-toast backdrop-blur"
      >
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

      <!-- 4. 標準プロジェクト -->
      <div class="mb-4 rounded-2xl border border-surface-border bg-surface-raised/80 p-4">
        <button
          type="button"
          class="flex w-full items-start justify-between gap-3 text-left"
          :aria-expanded="projectsExpanded"
          @click="projectsExpanded = !projectsExpanded"
        >
          <div class="min-w-0">
            <h3 class="font-display text-xs tracking-wide text-ink-muted">Conversions / Standard Projects</h3>
            <p v-if="!projectsExpanded" class="mt-1 text-xs text-ink-muted">タップで展開</p>
          </div>
          <ChevronDown v-if="!projectsExpanded" class="mt-0.5 h-4 w-4 shrink-0 text-ink-muted" />
          <ChevronUp v-else class="mt-0.5 h-4 w-4 shrink-0 text-ink-muted" />
        </button>
        <div v-show="projectsExpanded" class="mt-3">
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

      <!-- 5. 研究フェーズパネル — 自分の購入が完了したらアコーディオンで折りたたむ -->
      <section
        v-if="state.phase === 'RESEARCH'"
        class="mb-4 rounded-2xl border border-cyan-900/40 bg-cyan-950/20 p-4"
      >
        <button
          type="button"
          class="flex w-full items-start justify-between gap-3 text-left"
          :aria-expanded="researchExpanded"
          @click="researchExpanded = !researchExpanded"
        >
          <div class="min-w-0">
            <h2 class="font-display text-sm tracking-wide text-cyan-200">研究フェイズ — カード購入</h2>
            <p v-if="!researchExpanded" class="mt-1 text-xs text-ink-muted">
              {{ me.research_done ? '購入済み — タップで詳細' : 'タップで展開' }}
            </p>
          </div>
          <ChevronDown v-if="!researchExpanded" class="mt-0.5 h-4 w-4 shrink-0 text-cyan-200/70" />
          <ChevronUp v-else class="mt-0.5 h-4 w-4 shrink-0 text-cyan-200/70" />
        </button>
        <div v-show="researchExpanded" class="mt-3">
          <p class="text-sm text-ink-muted">
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
        </div>
      </section>

      <!-- 6. 資源パネル -->
      <div class="mb-4 rounded-2xl border border-surface-border bg-surface-raised/70 p-4">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 items-center gap-3">
            <span class="h-3.5 w-3.5 shrink-0 rounded-full" :style="{ backgroundColor: me.color }" />
            <p class="truncate font-semibold">
              {{ me.name }}
              <span v-if="me.passed" class="ml-1 text-xs text-red-300">PASS</span>
            </p>
          </div>
          <!-- 自分のTRは他参加者パネル(text-xs)より大きく、±も一回り大きくして目立たせる -->
          <div class="flex items-center gap-2">
            <div class="text-right leading-none">
              <p class="text-[10px] uppercase tracking-wider text-ink-muted">TR</p>
              <p class="font-display text-3xl font-bold tabular-nums">{{ me.tr }}</p>
            </div>
            <div class="flex gap-1">
              <RepeatPressButton
                v-for="d in [-1, 1]"
                :key="'tr' + d"
                class="rounded-lg bg-surface-border px-2.5 py-1.5 text-sm font-semibold hover:bg-mars"
                :disabled="!connected"
                @press="$emit('update', { target: 'tr', delta: d })"
              >
                {{ d > 0 ? '+' : '' }}{{ d }}
              </RepeatPressButton>
            </div>
          </div>
        </div>
        <div class="grid grid-cols-3 gap-2 sm:grid-cols-6">
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
      </div>

      <!-- 7. 他メンバー資源パネルなど -->
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
            @skip="(id) => $emit('skip-player', id)"
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
    </template>
  </div>

  <!-- モバイル: 1画面固定 + ボトムシート呼び出し型（デザイン案 1b/2a を現行トークンで実装） -->
  <div class="flex h-dvh w-full flex-col overflow-hidden font-body text-ink sm:hidden">
    <p v-if="!connected" class="flex-none bg-amber-950/50 px-3 py-1.5 text-center text-xs text-amber-200">
      再接続中です。オンラインになるまで操作できません。
    </p>

    <header class="flex flex-none items-center justify-between border-b border-surface-border bg-surface-raised/80 px-3 py-2 backdrop-blur">
      <div class="flex items-baseline gap-2">
        <span class="font-display text-[11px] tracking-[0.2em] text-ink-muted">GEN</span>
        <span class="font-display text-2xl font-bold leading-none">{{ state.generation }}</span>
        <span class="rounded-md bg-mars px-2 py-0.5 font-display text-[11px] tracking-wider text-white">{{ phaseLabel }}</span>
      </div>
      <div class="flex items-center gap-2.5">
        <button
          type="button"
          class="inline-flex items-center gap-1 rounded-md border border-surface-border bg-surface px-2 py-1 font-display text-xs tracking-widest text-ink"
          :class="{ 'border-emerald-600 text-emerald-300': roomCodeCopied }"
          @click="copyCode"
        >
          <Check v-if="roomCodeCopied" class="h-3 w-3" />
          <Copy v-else class="h-3 w-3" />
          {{ roomCode }}
        </button>
        <Wifi v-if="connected" class="h-4 w-4 text-emerald-400" />
        <WifiOff v-else class="h-4 w-4 text-red-400" />
      </div>
    </header>

    <!-- 終了/VP集計: デスクトップ同様、ENDED では VP ヘルパのみを表示 -->
    <div v-if="state.phase === 'ENDED'" class="min-h-0 flex-1 overflow-y-auto p-3">
      <VPHelper
        :players="orderedPlayers"
        :player-id="playerId"
        :score-fields="scoreFields"
        @score="(p) => $emit('score', p)"
      />
    </div>

    <template v-else>
      <!-- グローバルパラメータ: 常時表示ストリップ -->
      <div class="flex-none border-b border-surface-border px-3 py-2">
        <GlobalParamsBar
          compact
          :global-params="state.global_params"
          :is-host="isHost"
          @update-param="(p) => $emit('global-param', p)"
          @open-config="configModalOpen = true"
        />
      </div>

      <!-- 研究フェイズはカード購入、それ以外は手番/アクションバーを常時表示 -->
      <div class="flex-none border-b border-surface-border bg-surface-raised/90 px-3 py-2.5">
        <template v-if="state.phase === 'RESEARCH'">
          <div class="flex items-center justify-between gap-2">
            <p class="text-xs text-ink-muted">
              研究フェイズ — {{ useCardBuyKeypad ? '初期手札10枚' : '4枚ドロー' }}想定・3 MC/枚
            </p>
            <span v-if="me.research_done" class="shrink-0 text-xs font-semibold text-emerald-300">購入済み</span>
          </div>
          <div v-if="!me.research_done" class="mt-2 flex flex-wrap items-center gap-2">
            <button
              v-if="useCardBuyKeypad"
              type="button"
              class="rounded-lg border border-surface-border bg-surface px-4 py-1.5 font-display text-xl font-bold tabular-nums"
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
                class="h-8 w-8 rounded-full text-xs font-semibold transition"
                :class="cardBuy === n ? 'bg-cyan-600 text-white' : 'bg-surface text-ink-muted'"
                @click="cardBuy = n"
              >
                {{ n }}
              </button>
            </div>
            <span class="text-xs text-ink-muted">= {{ cardBuy * 3 }} MC</span>
            <button
              type="button"
              class="ml-auto rounded-xl bg-cyan-700 px-3 py-1.5 text-xs font-semibold text-white"
              @click="$emit('buy-cards', cardBuy)"
            >
              購入確定
            </button>
          </div>
        </template>
        <template v-else>
          <div class="flex items-center justify-between gap-2">
            <div class="min-w-0 text-xs text-ink-muted">
              <template v-if="state.phase === 'PRODUCTION_WAIT'">
                全員パス済み — 産出を実行
              </template>
              <template v-else>
                手番
                <strong class="text-sm font-semibold" :style="{ color: activePlayer?.color }">{{ activePlayer?.name || '—' }}</strong>
                <span class="ml-1">残 {{ actionsLeft }}/2</span>
              </template>
            </div>
            <div class="flex flex-none gap-1.5">
              <button
                v-if="isMyTurn && state.phase === 'ACTION'"
                type="button"
                class="inline-flex items-center gap-1.5 rounded-xl bg-mars px-3.5 py-2 text-xs font-semibold text-white animate-pulseReady"
                @click="$emit('claim-action')"
              >
                <Zap class="h-3.5 w-3.5" />
                実施
              </button>
              <button
                v-if="isMyTurn && hasActedThisTurn && state.phase === 'ACTION'"
                type="button"
                class="inline-flex items-center rounded-xl bg-surface-border px-2.5 py-2"
                title="ターン終了"
                @click="$emit('end-turn')"
              >
                <SkipForward class="h-3.5 w-3.5" />
              </button>
              <button
                v-if="isMyTurn && !hasActedThisTurn && state.phase === 'ACTION'"
                type="button"
                class="inline-flex items-center rounded-xl border border-red-800/50 bg-red-950/40 px-2.5 py-2 text-red-200"
                title="パス（世代脱落）"
                @click="$emit('pass')"
              >
                <Flag class="h-3.5 w-3.5" />
              </button>
              <button
                v-if="state.phase === 'PRODUCTION_WAIT'"
                type="button"
                class="rounded-xl bg-mars px-3.5 py-2 text-xs font-semibold text-white"
                @click="$emit('ready')"
              >
                産出実行
              </button>
            </div>
          </div>
        </template>
      </div>

      <!-- スクロール領域: 自分の資源 + Opponents 一覧（タップで詳細シート） -->
      <div class="min-h-0 flex-1 overflow-y-auto px-3 py-3">
        <div class="mb-2 flex items-end justify-between">
          <h4 class="font-display text-xs tracking-[0.12em] text-ink-muted">MY RESOURCES</h4>
          <div class="flex items-center gap-2">
            <span class="text-[10px] tracking-wider text-ink-muted">TR</span>
            <span class="font-display text-2xl font-bold leading-none tabular-nums">{{ me.tr }}</span>
            <RepeatPressButton
              v-for="d in [-1, 1]"
              :key="'mtr' + d"
              class="h-8 w-8 rounded-lg bg-surface-border text-sm font-semibold"
              :disabled="!connected"
              @press="$emit('update', { target: 'tr', delta: d })"
            >
              {{ d > 0 ? '+' : '' }}{{ d }}
            </RepeatPressButton>
          </div>
        </div>
        <div class="grid grid-cols-3 gap-2">
          <ResourceCard
            v-for="key in resourceOrder"
            :key="'m' + key"
            :resource-key="key"
            :meta="resourceMeta[key]"
            :stock="me.resources?.[key]?.stock ?? 0"
            :production="me.resources?.[key]?.production ?? 0"
            :interactive="connected"
            @change="(p) => $emit('update', p)"
          />
        </div>

        <h4 class="mb-1.5 mt-4 font-display text-xs tracking-[0.12em] text-ink-muted">OPPONENTS</h4>
        <div class="border-t border-surface-border">
          <button
            v-for="op in opponents"
            :key="'m' + op.id"
            type="button"
            class="flex w-full items-center gap-2 border-b border-surface-border py-2.5 text-left"
            @click="openOpponentSheet(op)"
          >
            <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ backgroundColor: op.color }" />
            <span class="min-w-0 flex-1 truncate font-semibold">
              {{ op.name }}
              <span v-if="op.passed" class="ml-1 text-[10px] text-red-300">PASS</span>
              <span v-if="op.online === false" class="ml-1 text-[10px] text-amber-300">オフライン</span>
            </span>
            <span class="font-display text-lg font-bold tabular-nums">{{ op.tr }}</span>
            <span class="text-[9px] tracking-wider text-ink-muted">TR</span>
            <ChevronRight class="h-4 w-4 shrink-0 text-ink-muted" />
          </button>
          <p v-if="!opponents.length" class="py-8 text-center text-xs text-ink-muted">
            他プレイヤーの参加を待っています…<br />
            コード <span class="font-display tracking-widest text-ink">{{ roomCode }}</span> を共有
          </p>
        </div>
      </div>

      <!-- ボトムナビ: 標準PJ / タグ / ログ / VP -->
      <div class="grid flex-none grid-cols-4 gap-1.5 border-t border-surface-border bg-surface-raised/90 p-2">
        <button
          type="button"
          class="flex flex-col items-center gap-0.5 rounded-lg border border-surface-border bg-surface py-2 text-[11px]"
          @click="openMobileSheet('projects')"
        >
          <Hammer class="h-[18px] w-[18px]" />
          標準PJ
        </button>
        <button
          type="button"
          class="flex flex-col items-center gap-0.5 rounded-lg border border-surface-border bg-surface py-2 text-[11px]"
          @click="openMobileSheet('tags')"
        >
          <Tags class="h-[18px] w-[18px]" />
          タグ
        </button>
        <button
          type="button"
          class="flex flex-col items-center gap-0.5 rounded-lg border border-surface-border bg-surface py-2 text-[11px]"
          @click="$emit('activity')"
        >
          <ScrollText class="h-[18px] w-[18px]" />
          ログ
        </button>
        <button
          type="button"
          class="flex flex-col items-center gap-0.5 rounded-lg border border-surface-border bg-surface py-2 text-[11px]"
          @click="openMobileSheet('vp')"
        >
          <Trophy class="h-[18px] w-[18px]" />
          VP
        </button>
      </div>
    </template>

    <!-- ボトムシート: 標準PJ / タグ / VP / 相手詳細 -->
    <div
      v-if="mobileSheet"
      class="fixed inset-0 z-40 flex items-end bg-black/60"
      @click.self="closeMobileSheet"
    >
      <div class="max-h-[75vh] w-full overflow-y-auto rounded-t-2xl border-t border-surface-border bg-surface-raised p-4 pb-6 shadow-toast">
        <div class="mb-3 flex items-center justify-between">
          <h4 class="font-display text-lg font-bold">{{ mobileSheetTitle }}</h4>
          <button type="button" class="rounded-lg border border-surface-border bg-surface p-1.5 text-ink-muted" @click="closeMobileSheet">
            <X class="h-4 w-4" />
          </button>
        </div>

        <template v-if="mobileSheet === 'projects'">
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
          <div class="mt-3 flex flex-col divide-y divide-surface-border overflow-hidden rounded-xl border border-surface-border">
            <button
              v-for="p in projects"
              :key="p.kind"
              type="button"
              class="flex items-center justify-between gap-2 bg-surface px-3 py-2.5 text-left disabled:opacity-40"
              :disabled="!isMyTurn"
              :title="p.cost"
              @click="onProject(p)"
            >
              <span class="font-display text-base">{{ p.label }}</span>
              <span class="text-xs text-ink-muted">{{ p.cost }}</span>
            </button>
          </div>
          <label class="mt-3 block text-xs text-ink-muted">
            特許売却の枚数
            <input v-model.number="sellCount" type="number" min="1" max="20" class="ml-2 w-16 rounded border border-surface-border bg-surface px-2 py-1 text-ink" />
          </label>
          <p v-if="state.phase === 'ACTION' && !isMyTurn" class="mt-2 text-xs text-ink-muted">
            標準プロジェクト／変換は自分の手番のみ（資源の手動調整はいつでも可）
          </p>
        </template>

        <template v-if="mobileSheet === 'tags'">
          <div class="grid grid-cols-3 gap-2">
            <div v-for="tag in tags" :key="'ms' + tag" class="rounded-lg bg-surface px-1.5 py-2 text-center">
              <p class="truncate text-[10px] text-ink-muted">{{ tag }}</p>
              <p class="font-display text-lg tabular-nums">{{ me.tags?.[tag] || 0 }}</p>
              <div class="mt-1 flex justify-center gap-1">
                <RepeatPressButton
                  class="rounded bg-surface-border px-2 py-1 text-xs"
                  :disabled="!connected"
                  @press="$emit('tag', { tag, delta: -1 })"
                >
                  −
                </RepeatPressButton>
                <RepeatPressButton
                  class="rounded bg-surface-border px-2 py-1 text-xs"
                  :disabled="!connected"
                  @press="$emit('tag', { tag, delta: 1 })"
                >
                  +
                </RepeatPressButton>
              </div>
            </div>
          </div>
        </template>

        <template v-if="mobileSheet === 'vp'">
          <VPHelper
            embedded
            :players="orderedPlayers"
            :player-id="playerId"
            :score-fields="scoreFields"
            @score="(p) => $emit('score', p)"
          />
        </template>

        <template v-if="mobileSheet === 'opponent' && mobileSheetOpponent">
          <OpponentCard
            :player="mobileSheetOpponent"
            :resource-order="resourceOrder"
            :resource-meta="resourceMeta"
            :active="state.active_player_id === mobileSheetOpponent.id"
            @skip="(id) => { closeMobileSheet(); $emit('skip-player', id) }"
          />
        </template>
      </div>
    </div>
  </div>

  <!-- Global Parameters Config Modal（デスクトップ/モバイル共通・fixed オーバーレイ） -->
  <GlobalParamConfigModal
    :open="configModalOpen"
    :is-host="isHost"
    :global-params="state.global_params"
    @save="(p) => $emit('configure-global-params', p)"
    @close="configModalOpen = false"
  />
</template>

