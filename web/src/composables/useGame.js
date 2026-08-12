import { computed, onUnmounted, reactive, ref, shallowRef } from 'vue'

const STORAGE_KEY = 'tbg-rse-session'

const RESOURCE_ORDER = ['MC', 'Steel', 'Titanium', 'Plant', 'Energy', 'Heat']

const RESOURCE_META = {
  MC: { label: 'MC', short: 'MC', accent: '#e0b84a' },
  Steel: { label: 'Steel', short: 'St', accent: '#9aa4b2' },
  Titanium: { label: 'Titanium', short: 'Ti', accent: '#6eb0d6' },
  Plant: { label: 'Plant', short: 'Pl', accent: '#5cbc6a' },
  Energy: { label: 'Energy', short: 'En', accent: '#e8d24a' },
  Heat: { label: 'Heat', short: 'Ht', accent: '#e06a3a' },
}

const TAGS = [
  'Building', 'Space', 'Science', 'Power', 'Earth', 'Jovian',
  'Plant', 'Microbe', 'Animal', 'City', 'Wild', 'Event',
]

const STANDARD_PROJECTS = [
  { kind: 'sell_patents', label: '特許売却', cost: '1MC/枚', needsCards: true },
  { kind: 'power_plant', label: '発電所', cost: '11 MC' },
  { kind: 'asteroid', label: '小惑星', cost: '14 MC' },
  { kind: 'aquifer', label: '帯水層', cost: '18 MC' },
  { kind: 'greenery_project', label: '緑化(標準)', cost: '23 MC' },
  { kind: 'city', label: '都市', cost: '25 MC' },
]

const SCORE_FIELDS = [
  { field: 'greenery_tiles', label: '緑化タイル' },
  { field: 'city_tiles', label: '都市タイル' },
  { field: 'city_adj_greenery', label: '都市隣接緑化VP' },
  { field: 'milestone', label: 'マイルストーン' },
  { field: 'award', label: '賞' },
  { field: 'cards', label: 'カードVP' },
  { field: 'other', label: 'その他' },
]

const COLORS = ['#FF5555', '#55AAFF', '#55FF88', '#FFAA33', '#C77DFF', '#FF77AA']
const SEATS = [1, 2, 3, 4, 5]

function loadSession() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

function saveSession(session) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(session))
}

function clearSession() {
  localStorage.removeItem(STORAGE_KEY)
}

function wsURL() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/ws`
}

function totalVP(player) {
  if (!player) return 0
  const s = player.score || {}
  return (
    (player.tr || 0) +
    (s.greenery_tiles || 0) +
    (s.city_tiles || 0) +
    (s.city_adj_greenery || 0) +
    (s.milestone || 0) +
    (s.award || 0) +
    (s.cards || 0) +
    (s.other || 0)
  )
}

export function useGame() {
  const connected = ref(false)
  const error = ref('')
  const roomCode = ref('')
  const playerId = ref('')
  const hostPlayerId = ref('')
  const reconnectToken = ref('')
  const state = shallowRef(null)
  const toasts = ref([])
  const activityOpen = ref(false)
  const activity = ref([])
  const lastHighlight = reactive({})

  const takenColors = ref([])
  const takenSeats = ref([])

  let ws = null
  let toastSeq = 0
  let reconnectTimer = null
  let intentionalClose = false

  const me = computed(() => state.value?.players?.[playerId.value] ?? null)
  const opponents = computed(() => {
    if (!state.value?.players) return []
    const order = state.value.turn_order || []
    const list = Object.values(state.value.players).filter((p) => p.id !== playerId.value)
    if (!order.length) return list
    return [...list].sort((a, b) => order.indexOf(a.id) - order.indexOf(b.id))
  })
  const isHost = computed(() => playerId.value && playerId.value === hostPlayerId.value)
  const isMyTurn = computed(
    () => state.value?.phase === 'ACTION' && state.value?.active_player_id === playerId.value && !me.value?.passed,
  )
  const activePlayer = computed(() => {
    const id = state.value?.active_player_id
    return id ? state.value?.players?.[id] : null
  })
  const orderedPlayers = computed(() => {
    if (!state.value?.players) return []
    const order = state.value.turn_order || []
    if (!order.length) return Object.values(state.value.players)
    return order.map((id) => state.value.players[id]).filter(Boolean)
  })

  function send(action, payload = {}) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      error.value = 'Not connected'
      return
    }
    ws.send(JSON.stringify({ action, payload }))
  }

  function pushToast(playerName, message, timestamp) {
    const id = ++toastSeq
    toasts.value = [{ id, playerName, message, timestamp }, ...toasts.value].slice(0, 5)
    activity.value = [{ id, playerName, message, timestamp }, ...activity.value].slice(0, 100)
    setTimeout(() => {
      toasts.value = toasts.value.filter((t) => t.id !== id)
    }, 4200)
  }

  function handleEvent(msg) {
    switch (msg.event) {
      case 'ROOM_JOINED': {
        const p = msg.payload
        roomCode.value = p.room_code
        playerId.value = p.player_id
        reconnectToken.value = p.reconnect_token
        hostPlayerId.value = p.host_player_id || ''
        state.value = p.state
        saveSession({
          room_code: p.room_code,
          player_id: p.player_id,
          reconnect_token: p.reconnect_token,
          host_player_id: hostPlayerId.value,
        })
        error.value = ''
        break
      }
      case 'STATE_UPDATE':
        state.value = msg.payload
        break
      case 'NOTIFICATION': {
        const n = msg.payload
        pushToast(n.player_name, n.message, n.timestamp)
        if (state.value?.players) {
          const hit = Object.values(state.value.players).find((pl) => pl.name === n.player_name)
          if (hit) lastHighlight[hit.id] = Date.now()
        }
        break
      }
      case 'ROOM_INFO': {
        const p = msg.payload || {}
        takenColors.value = Array.isArray(p.taken_colors) ? p.taken_colors : []
        takenSeats.value = Array.isArray(p.taken_seats) ? p.taken_seats : []
        if (p.found === false) {
          error.value = 'room not found'
        } else if (!state.value) {
          error.value = ''
        }
        break
      }
      case 'ERROR': {
        error.value = msg.payload?.message || 'Error'
        if (!state.value) clearSession()
        break
      }
      default:
        break
    }
  }

  function connect(onOpen) {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    // Detach handlers before close — browsers fire error/close on the dying socket.
    if (ws) {
      const prev = ws
      prev.onopen = null
      prev.onclose = null
      prev.onerror = null
      prev.onmessage = null
      try {
        prev.close()
      } catch {
        /* ignore */
      }
    }
    intentionalClose = false
    const socket = new WebSocket(wsURL())
    ws = socket
    socket.onopen = () => {
      if (ws !== socket) return
      connected.value = true
      error.value = ''
      onOpen?.()
    }
    socket.onclose = () => {
      if (ws !== socket) return
      connected.value = false
      if (!intentionalClose) scheduleReconnect()
    }
    socket.onerror = () => {
      if (ws !== socket) return
      error.value = 'WebSocket error'
    }
    socket.onmessage = (ev) => {
      if (ws !== socket) return
      try {
        handleEvent(JSON.parse(ev.data))
      } catch (e) {
        console.error(e)
      }
    }
  }

  function scheduleReconnect() {
    if (reconnectTimer) return
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      const session = loadSession()
      if (!session) {
        connect()
        return
      }
      connect(() => {
        send('RECONNECT', {
          room_code: session.room_code,
          player_id: session.player_id,
          reconnect_token: session.reconnect_token,
        })
      })
    }, 1200)
  }

  function createRoom(name, color, seat = 1) {
    connect(() => send('CREATE_ROOM', { name, color, seat }))
  }

  function joinRoom(code, name, color, seat = 0) {
    connect(() => send('JOIN_ROOM', { room_code: code, name, color, seat }))
  }

  function peekRoom(code) {
    const normalized = (code || '').trim().toUpperCase()
    if (normalized.length < 4) {
      takenColors.value = []
      takenSeats.value = []
      return
    }
    const doPeek = () => send('PEEK_ROOM', { room_code: normalized })
    if (ws && ws.readyState === WebSocket.OPEN) {
      doPeek()
      return
    }
    connect(doPeek)
  }

  function tryAutoReconnect() {
    const session = loadSession()
    if (!session?.room_code) return false
    hostPlayerId.value = session.host_player_id || ''
    connect(() => {
      send('RECONNECT', {
        room_code: session.room_code,
        player_id: session.player_id,
        reconnect_token: session.reconnect_token,
      })
    })
    return true
  }

  function leaveLocal() {
    intentionalClose = true
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    clearSession()
    roomCode.value = ''
    playerId.value = ''
    hostPlayerId.value = ''
    reconnectToken.value = ''
    state.value = null
    takenColors.value = []
    takenSeats.value = []
    if (ws) {
      ws.onclose = null
      ws.close()
      ws = null
    }
    connected.value = false
  }

  onUnmounted(() => {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    if (ws) {
      ws.onclose = null
      ws.close()
    }
  })

  return {
    RESOURCE_ORDER,
    RESOURCE_META,
    TAGS,
    STANDARD_PROJECTS,
    SCORE_FIELDS,
    COLORS,
    SEATS,
    totalVP,
    connected,
    error,
    roomCode,
    playerId,
    hostPlayerId,
    state,
    me,
    opponents,
    orderedPlayers,
    isHost,
    isMyTurn,
    activePlayer,
    toasts,
    activityOpen,
    activity,
    lastHighlight,
    takenColors,
    takenSeats,
    createRoom,
    joinRoom,
    peekRoom,
    tryAutoReconnect,
    leaveLocal,
    updateResource: (target, resource, delta) => send('UPDATE_RESOURCE', { target, resource, delta }),
    buyCards: (count) => send('BUY_CARDS', { count }),
    standardProject: (kind, cardsSold = 0) => send('STANDARD_PROJECT', { kind, cards_sold: cardsSold }),
    shortcut: (kind) => send('SHORTCUT', { kind }),
    endTurn: () => send('END_TURN', {}),
    pass: () => send('PASS', {}),
    claimAction: () => send('CLAIM_ACTION', {}),
    readyProduction: () => send('READY_PRODUCTION', {}),
    updateTag: (tag, delta) => send('UPDATE_TAG', { tag, delta }),
    updateScore: (field, delta) => send('UPDATE_SCORE', { field, delta }),
    endGame: () => send('END_GAME', {}),
  }
}
