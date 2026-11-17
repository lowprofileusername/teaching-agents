// Gesture classification borrowed from MaClaw pet_click.go: a 5px drag
// threshold separates drag from click; click bursts are adjudicated after the
// double-click interval (single opens the workbench, double/multi only react).
export const DRAG_THRESHOLD_PX = 5;
export const HOVER_COOLDOWN_MS = 5000;
export const CLICK_BURST_MS = 500;
export function isDrag(dx, dy) { return Math.hypot(dx, dy) > DRAG_THRESHOLD_PX; }
export function classifyClick(burst) { return burst >= 3 ? 'multi' : burst === 2 ? 'double' : 'single'; }
export function clickCue(cls) { return cls === 'single' ? 'greet' : cls === 'double' ? 'notice' : 'celebrate'; }
export function clickOpensMain(cls) { return cls === 'single'; }
// Hover greets at most once per cooldown (MaClaw petHoverCooldown), and never
// stops the pet walking.
export function hoverDue(lastAt, now) { return now - lastAt >= HOVER_COOLDOWN_MS; }