import { onUnmounted } from 'vue'

const HOLD_DELAY_MS = 400
const REPEAT_INTERVAL_MS = 150
const CLICK_SUPPRESS_MS = 500

export function useRepeatPress(onRepeat) {
  let holdTimer = null
  let repeatTimer = null
  let lastPointerFire = 0

  function clearTimers() {
    if (holdTimer) {
      clearTimeout(holdTimer)
      holdTimer = null
    }
    if (repeatTimer) {
      clearInterval(repeatTimer)
      repeatTimer = null
    }
  }

  function fire() {
    onRepeat()
  }

  function onPointerDown(e) {
    if (e.button !== 0) return
    const el = e.currentTarget
    if (el?.disabled) return

    e.preventDefault()
    el?.setPointerCapture?.(e.pointerId)

    lastPointerFire = Date.now()
    fire()
    clearTimers()
    holdTimer = setTimeout(() => {
      holdTimer = null
      repeatTimer = setInterval(fire, REPEAT_INTERVAL_MS)
    }, HOLD_DELAY_MS)
  }

  function onPointerUp(e) {
    e?.currentTarget?.releasePointerCapture?.(e.pointerId)
    clearTimers()
  }

  function onClick(e) {
    if (Date.now() - lastPointerFire < CLICK_SUPPRESS_MS) {
      e.preventDefault()
      return
    }
    fire()
  }

  onUnmounted(clearTimers)

  return { onPointerDown, onPointerUp, onClick }
}
