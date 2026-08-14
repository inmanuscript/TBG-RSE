<script setup>
import { onMounted } from 'vue'
import { useGame } from './composables/useGame'
import Lobby from './components/Lobby.vue'
import Dashboard from './components/Dashboard.vue'
import ToastFeed from './components/ToastFeed.vue'
import ActivityLog from './components/ActivityLog.vue'

const game = useGame()

const {
  RESOURCE_ORDER,
  RESOURCE_META,
  TAGS,
  STANDARD_PROJECTS,
  SCORE_FIELDS,
  COLORS,
  SEATS,
  connected,
  lobbyPending,
  error,
  roomCode,
  playerId,
  state,
  me,
  opponents,
  orderedPlayers,
  isMyTurn,
  activePlayer,
  isHost,
  toasts,
  activityOpen,
  activity,
  lastHighlight,
  takenColors,
  takenSeats,
  roomPlayers,
  allowNewJoin,
  createRoom,
  joinRoom,
  peekRoom,
  tryAutoReconnect,
  leaveLocal,
  updateResource,
  buyCards,
  standardProject,
  shortcut,
  endTurn,
  pass,
  claimAction,
  readyProduction,
  updateTag,
  updateScore,
  updateGlobalParam,
  configureGlobalParams,
  clearError,
  dismissToast,
  endGame,
} = game

onMounted(() => {
  tryAutoReconnect()
})

function onCreate({ name, color, seat }) {
  createRoom(name, color, seat)
}

function onJoin({ name, color, seat, roomCode: code, reclaimPlayerId }) {
  joinRoom(code, name, color, seat, reclaimPlayerId || '')
}

function onUpdate(payload) {
  if (payload.target === 'tr') {
    updateResource('tr', '', payload.delta)
  } else {
    updateResource(payload.target, payload.resource, payload.delta)
  }
}

function onLeave() {
  if (confirm('ローカルセッションをクリアしてロビーに戻りますか？（サーバー上の席は残ります）')) {
    leaveLocal()
  }
}

function onEndGame() {
  if (confirm('ゲームを終了して VP ヘルパを開きますか？')) {
    endGame()
  }
}
</script>

<template>
  <ToastFeed :toasts="toasts" @dismiss="dismissToast" />
  <ActivityLog
    :open="activityOpen"
    :items="activity"
    :players="orderedPlayers"
    :resource-meta="RESOURCE_META"
    :my-player-id="playerId"
    @close="activityOpen = false"
  />

  <Dashboard
    v-if="state && me"
    :state="state"
    :me="me"
    :opponents="opponents"
    :ordered-players="orderedPlayers"
    :room-code="roomCode"
    :connected="connected"
    :resource-order="RESOURCE_ORDER"
    :resource-meta="RESOURCE_META"
    :tags="TAGS"
    :projects="STANDARD_PROJECTS"
    :score-fields="SCORE_FIELDS"
    :last-highlight="lastHighlight"
    :is-my-turn="isMyTurn"
    :active-player="activePlayer"
    :player-id="playerId"
    :is-host="isHost"
    :error="error"
    @update="onUpdate"
    @ready="readyProduction()"
    @shortcut="shortcut"
    @project="(p) => standardProject(p.kind, p.cardsSold)"
    @buy-cards="buyCards"
    @claim-action="claimAction()"
    @end-turn="endTurn()"
    @pass="pass()"
    @tag="(p) => updateTag(p.tag, p.delta)"
    @score="(p) => updateScore(p.field, p.delta)"
    @global-param="(p) => updateGlobalParam(p.paramId, p.deltaSteps, p.grantTR)"
    @configure-global-params="configureGlobalParams"
    @clear-error="clearError"
    @end-game="onEndGame"
    @activity="activityOpen = true"
    @leave="onLeave"
  />
  <Lobby
    v-else
    :colors="COLORS"
    :seats="SEATS"
    :taken-colors="takenColors"
    :taken-seats="takenSeats"
    :room-players="roomPlayers"
    :allow-new-join="allowNewJoin"
    :error="error"
    :connecting="lobbyPending"
    @create="onCreate"
    @join="onJoin"
    @peek="peekRoom"
  />
</template>
