<script setup>
import { computed, ref, watch } from 'vue'
import { Rocket, Users, LogIn, RotateCcw } from '@lucide/vue'

const props = defineProps({
  colors: { type: Array, required: true },
  seats: { type: Array, default: () => [1, 2, 3, 4, 5] },
  takenColors: { type: Array, default: () => [] },
  takenSeats: { type: Array, default: () => [] },
  roomPlayers: { type: Array, default: () => [] },
  allowNewJoin: { type: Boolean, default: true },
  error: { type: String, default: '' },
  connecting: { type: Boolean, default: false },
})

const emit = defineEmits(['create', 'join', 'peek'])

const mode = ref('create')
const joinSubMode = ref('new')
const name = ref('')
const color = ref(props.colors[0])
const seat = ref(1)
const roomCode = ref('')
const reclaimPlayerId = ref('')

const takenColorSet = computed(() => {
  const set = new Set()
  for (const c of props.takenColors || []) set.add(String(c).toUpperCase())
  return set
})
const takenSeatSet = computed(() => new Set((props.takenSeats || []).map(Number)))

const offlinePlayers = computed(() =>
  (props.roomPlayers || [])
    .filter((p) => !p.online)
    .sort((a, b) => Number(a.seat) - Number(b.seat)),
)

const roomKnown = computed(() => props.roomPlayers.length > 0 || (props.takenSeats?.length ?? 0) > 0)

function isColorTaken(c) {
  return mode.value === 'join' && joinSubMode.value === 'new' && takenColorSet.value.has(String(c).toUpperCase())
}
function isSeatTaken(s) {
  return mode.value === 'join' && joinSubMode.value === 'new' && takenSeatSet.value.has(Number(s))
}

function pickDefaults() {
  const freeColor = props.colors.find((c) => !isColorTaken(c))
  if (freeColor) color.value = freeColor
  const freeSeat = props.seats.find((s) => !isSeatTaken(s))
  if (freeSeat != null) seat.value = freeSeat
}

function selectReclaimPlayer(id) {
  reclaimPlayerId.value = id
  const player = props.roomPlayers.find((p) => p.id === id)
  if (player) {
    name.value = player.name
    seat.value = player.seat
    color.value = player.color
  }
}

watch(mode, (m) => {
  if (m === 'create') {
    color.value = props.colors[0]
    seat.value = 1
    reclaimPlayerId.value = ''
    joinSubMode.value = 'new'
  } else {
    pickDefaults()
    if (roomCode.value.trim().length >= 4) emit('peek', roomCode.value.trim())
  }
})

watch(() => props.allowNewJoin, (allow) => {
  if (mode.value !== 'join') return
  if (!allow) joinSubMode.value = 'reclaim'
})

watch([() => props.takenColors, () => props.takenSeats], () => {
  if (mode.value !== 'join' || joinSubMode.value !== 'new') return
  if (isColorTaken(color.value) || isSeatTaken(seat.value)) pickDefaults()
})

watch(offlinePlayers, (list) => {
  if (mode.value !== 'join' || joinSubMode.value !== 'reclaim') return
  if (!list.length) {
    reclaimPlayerId.value = ''
    return
  }
  if (!list.some((p) => p.id === reclaimPlayerId.value)) {
    selectReclaimPlayer(list[0].id)
  }
})

watch(joinSubMode, (sub) => {
  if (sub === 'new') {
    reclaimPlayerId.value = ''
    pickDefaults()
  } else if (offlinePlayers.value.length) {
    selectReclaimPlayer(offlinePlayers.value[0].id)
  }
})

let peekTimer = null
watch(roomCode, (code) => {
  if (mode.value !== 'join') return
  clearTimeout(peekTimer)
  reclaimPlayerId.value = ''
  peekTimer = setTimeout(() => emit('peek', code.trim()), 280)
})

const submitDisabled = computed(() => {
  if (props.connecting) return true
  if (mode.value !== 'join') return false
  if (joinSubMode.value === 'new') {
    return isColorTaken(color.value) || isSeatTaken(seat.value)
  }
  return !reclaimPlayerId.value || offlinePlayers.value.length === 0
})

function submit() {
  if (!name.value.trim()) return
  if (mode.value === 'create') {
    emit('create', { name: name.value.trim(), color: color.value, seat: seat.value })
  } else {
    if (!roomCode.value.trim()) return
    if (joinSubMode.value === 'new') {
      if (isColorTaken(color.value) || isSeatTaken(seat.value)) return
      emit('join', {
        name: name.value.trim(),
        color: color.value,
        seat: seat.value,
        roomCode: roomCode.value.trim().toUpperCase(),
      })
    } else {
      if (!reclaimPlayerId.value) return
      emit('join', {
        name: name.value.trim(),
        color: color.value,
        seat: seat.value,
        roomCode: roomCode.value.trim().toUpperCase(),
        reclaimPlayerId: reclaimPlayerId.value,
      })
    }
  }
}
</script>

<template>
  <div class="mx-auto flex min-h-screen max-w-lg flex-col justify-center px-4 py-10">
    <header class="mb-10 text-center animate-fadeUp">
      <p class="font-display text-xs tracking-[0.35em] text-mars-glow">TERRAFORMING MARS</p>
      <h1 class="mt-3 font-display text-3xl font-bold tracking-wide text-ink sm:text-4xl">
        TBG-RSE
      </h1>
      <p class="mt-3 text-sm text-ink-muted">
        資源・産出・世代進行をリアルタイム同期
      </p>
    </header>

    <div class="rounded-2xl border border-surface-border bg-surface-raised/90 p-6 shadow-toast backdrop-blur">
      <div class="mb-6 grid grid-cols-2 gap-2 rounded-xl bg-surface p-1">
        <button
          type="button"
          class="flex items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition"
          :class="mode === 'create' ? 'bg-mars text-white' : 'text-ink-muted hover:text-ink'"
          @click="mode = 'create'"
        >
          <Rocket class="h-4 w-4" />
          ルーム作成
        </button>
        <button
          type="button"
          class="flex items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition"
          :class="mode === 'join' ? 'bg-mars text-white' : 'text-ink-muted hover:text-ink'"
          @click="mode = 'join'"
        >
          <LogIn class="h-4 w-4" />
          参加
        </button>
      </div>

      <form class="space-y-4" @submit.prevent="submit">
        <label class="block">
          <span class="mb-1.5 block text-xs font-medium uppercase tracking-wider text-ink-muted">名前</span>
          <input
            v-model="name"
            class="w-full rounded-xl border border-surface-border bg-surface px-4 py-3 text-ink outline-none ring-mars focus:ring-2"
            placeholder="プレイヤー名"
            maxlength="24"
            required
          />
        </label>

        <label v-if="mode === 'join'" class="block">
          <span class="mb-1.5 block text-xs font-medium uppercase tracking-wider text-ink-muted">ルームコード</span>
          <input
            v-model="roomCode"
            class="w-full rounded-xl border border-surface-border bg-surface px-4 py-3 font-display tracking-[0.2em] text-ink outline-none ring-mars focus:ring-2"
            placeholder="ABC123"
            maxlength="8"
            required
          />
        </label>

        <template v-if="mode === 'join' && roomKnown">
          <div v-if="allowNewJoin" class="grid grid-cols-2 gap-2 rounded-xl bg-surface p-1">
            <button
              type="button"
              class="rounded-lg px-3 py-2 text-sm font-medium transition"
              :class="joinSubMode === 'new' ? 'bg-mars text-white' : 'text-ink-muted hover:text-ink'"
              @click="joinSubMode = 'new'"
            >
              新規参加
            </button>
            <button
              type="button"
              class="flex items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium transition"
              :class="joinSubMode === 'reclaim' ? 'bg-mars text-white' : 'text-ink-muted hover:text-ink'"
              @click="joinSubMode = 'reclaim'"
            >
              <RotateCcw class="h-3.5 w-3.5" />
              席を復帰
            </button>
          </div>
          <p v-else class="rounded-lg bg-surface px-3 py-2 text-sm text-ink-muted">
            ゲーム進行中のため、切断中のプレイヤー席のみ復帰できます。
          </p>
        </template>

        <div v-if="mode === 'join' && joinSubMode === 'reclaim' && roomKnown">
          <span class="mb-1.5 block text-xs font-medium uppercase tracking-wider text-ink-muted">切断中のプレイヤー</span>
          <div v-if="offlinePlayers.length" class="space-y-2">
            <button
              v-for="p in offlinePlayers"
              :key="p.id"
              type="button"
              class="flex w-full items-center gap-3 rounded-xl border px-3 py-2.5 text-left transition"
              :class="reclaimPlayerId === p.id ? 'border-mars bg-mars/10' : 'border-surface-border bg-surface hover:border-mars/50'"
              @click="selectReclaimPlayer(p.id)"
            >
              <span class="h-4 w-4 shrink-0 rounded-full" :style="{ backgroundColor: p.color }" />
              <span class="min-w-0 flex-1 truncate font-medium">{{ p.name }}</span>
              <span class="text-xs text-ink-muted">席 {{ p.seat }}</span>
            </button>
          </div>
          <p v-else class="rounded-lg bg-surface px-3 py-2 text-sm text-amber-200/90">
            切断中のプレイヤーがいません。全員オンラインの場合は、元の端末から再接続してください。
          </p>
        </div>

        <div v-if="mode !== 'join' || joinSubMode === 'new'">
          <span class="mb-1.5 block text-xs font-medium uppercase tracking-wider text-ink-muted">座席（時計回り）</span>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="s in seats"
              :key="s"
              type="button"
              class="h-10 w-10 rounded-xl border text-sm font-display transition"
              :disabled="isSeatTaken(s)"
              :class="[
                seat === s ? 'border-mars bg-mars text-white' : 'border-surface-border bg-surface text-ink',
                isSeatTaken(s) ? 'opacity-30 cursor-not-allowed line-through' : 'hover:border-mars',
              ]"
              @click="seat = s"
            >
              {{ s }}
            </button>
          </div>
          <p class="mt-1.5 text-xs text-ink-muted">席1→2→…の順がターン順になります</p>
        </div>

        <div v-if="mode !== 'join' || joinSubMode === 'new'">
          <span class="mb-1.5 block text-xs font-medium uppercase tracking-wider text-ink-muted">カラー</span>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="c in colors"
              :key="c"
              type="button"
              class="relative h-9 w-9 rounded-full border-2 transition"
              :style="{ backgroundColor: c }"
              :disabled="isColorTaken(c)"
              :class="[
                color === c ? 'border-white scale-110' : 'border-transparent',
                isColorTaken(c) ? 'opacity-25 cursor-not-allowed' : 'opacity-70 hover:opacity-100',
              ]"
              @click="color = c"
            >
              <span
                v-if="isColorTaken(c)"
                class="absolute inset-0 flex items-center justify-center text-[10px] font-bold text-white drop-shadow"
              >×</span>
            </button>
          </div>
        </div>

        <p v-if="error" class="rounded-lg bg-red-950/50 px-3 py-2 text-sm text-red-300">{{ error }}</p>

        <button
          type="submit"
          class="flex w-full items-center justify-center gap-2 rounded-xl bg-mars px-4 py-3.5 font-semibold text-white transition hover:bg-mars-glow disabled:hover:bg-mars"
          :disabled="submitDisabled"
        >
          <Users class="h-4 w-4" />
          {{
            mode === 'create'
              ? 'ルームを作成'
              : joinSubMode === 'reclaim'
                ? '席を復帰'
                : 'ルームに参加'
          }}
        </button>
      </form>
    </div>
  </div>
</template>
