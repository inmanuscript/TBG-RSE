import { onUnmounted } from 'vue'

const HOLD_DELAY_MS = 400
const REPEAT_INTERVAL_MS = 150
const CLICK_SUPPRESS_MS = 500
// How far a touch can travel before we treat the gesture as a scroll/drag
// instead of a tap (same idea as Android/iOS touch-slop). Prevents a page
// drag-scroll that happens to start on top of a +/- button from also
// bumping the resource count.
const MOVE_CANCEL_PX = 10

export function useRepeatPress(onRepeat) {
  let holdTimer = null
  let repeatTimer = null
  let lastPointerFire = 0

  // Touch only: unlike mouse, a touch pointerdown can't be trusted as "the
  // user meant to press this button" — it might be the start of a page
  // scroll. So for touch we defer the first fire() until either the finger
  // lifts without much movement (a tap) or the hold delay elapses without
  // much movement (a long-press-to-repeat). Mouse/pen keep firing instantly
  // on pointerdown as before.
  let trackingTouch = false
  let startX = 0
  let startY = 0
  let cancelled = false

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
    lastPointerFire = Date.now()
    onRepeat()
  }

  function onPointerDown(e) {
    if (e.button !== 0) return
    const el = e.currentTarget
    if (el?.disabled) return

    if (e.pointerType === 'touch') {
      trackingTouch = true
      cancelled = false
      startX = e.clientX
      startY = e.clientY
      // Arm the hold timer now so long-press-to-repeat still starts
      // counting from touch-down, but don't fire or capture the pointer
      // yet — a scroll gesture must still be free to take over.
      clearTimers()
      holdTimer = setTimeout(() => {
        holdTimer = null
        if (cancelled) return
        el?.setPointerCapture?.(e.pointerId)
        fire()
        repeatTimer = setInterval(fire, REPEAT_INTERVAL_MS)
      }, HOLD_DELAY_MS)
      return
    }

    trackingTouch = false
    e.preventDefault()
    el?.setPointerCapture?.(e.pointerId)
    clearTimers()
    fire()
    holdTimer = setTimeout(() => {
      holdTimer = null
      repeatTimer = setInterval(fire, REPEAT_INTERVAL_MS)
    }, HOLD_DELAY_MS)
  }

  function onPointerMove(e) {
    if (!trackingTouch || cancelled) return
    const dx = e.clientX - startX
    const dy = e.clientY - startY
    if (Math.hypot(dx, dy) > MOVE_CANCEL_PX) {
      // The finger has traveled — this is a scroll/drag, not a tap. Let
      // the browser handle it natively; don't fire anything for it.
      cancelled = true
      clearTimers()
    }
  }

  function onPointerUp(e) {
    e?.currentTarget?.releasePointerCapture?.(e.pointerId)
    // Only a genuine pointerup (not pointercancel/pointerleave) released
    // before the hold delay, with no cancelling movement, counts as a tap.
    if (trackingTouch && !cancelled && holdTimer && e.type === 'pointerup') {
      e.preventDefault()
      fire()
    }
    trackingTouch = false
    cancelled = false
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

  return { onPointerDown, onPointerMove, onPointerUp, onClick }
}
