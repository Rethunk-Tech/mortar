// Clips a full-window layer around the frameless caption buttons so they stay visible and clickable above it.
export const controlsCutout =
  'polygon(0 0, calc(100% - var(--window-controls)) 0, calc(100% - var(--window-controls)) var(--title-bar), 100% var(--title-bar), 100% 100%, 0 100%)'
