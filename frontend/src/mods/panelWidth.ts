const PANEL_DEFAULT_PX = 360
const PANEL_MIN_PX = 300
const PANEL_MAX_SHARE = 0.5
const KEY_STEP_PX = 16
const isWidth = (v: unknown): v is number => typeof v === 'number' && Number.isFinite(v)

const clampPanelWidth = (px: number, available: number) =>
  Math.max(PANEL_MIN_PX, Math.min(px, available * PANEL_MAX_SHARE))

export { clampPanelWidth, isWidth, KEY_STEP_PX, PANEL_DEFAULT_PX, PANEL_MIN_PX }
