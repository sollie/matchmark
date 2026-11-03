const TICK_MS = 50;
const SPEED_PX_PER_S = 25;
const END_PAUSE_MS = 4_000;
const IDLE_RESUME_MS = 10_000;
const INPUT_EVENTS = ["wheel", "touchstart", "keydown", "pointerdown"] as const;

export function autoScroll(node: HTMLElement) {
  let position = 0;
  let direction = 1;
  let pausedUntil = Date.now() + END_PAUSE_MS;

  const reducedMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)");

  const pauseForInput = () => {
    pausedUntil = Date.now() + IDLE_RESUME_MS;
  };
  for (const type of INPUT_EVENTS) node.addEventListener(type, pauseForInput, { passive: true });

  const timer = setInterval(() => {
    const max = node.scrollHeight - node.clientHeight;
    if (max <= 0 || reducedMotion?.matches) {
      if (max <= 0) node.scrollTop = 0;
      position = node.scrollTop;
      return;
    }
    if (Date.now() < pausedUntil) {
      position = node.scrollTop;
      return;
    }
    position = Math.min(max, Math.max(0, position + (direction * SPEED_PX_PER_S * TICK_MS) / 1000));
    node.scrollTop = position;
    if (position >= max || position <= 0) {
      direction = position >= max ? -1 : 1;
      pausedUntil = Date.now() + END_PAUSE_MS;
    }
  }, TICK_MS);

  return {
    destroy() {
      clearInterval(timer);
      for (const type of INPUT_EVENTS) node.removeEventListener(type, pauseForInput);
    },
  };
}
