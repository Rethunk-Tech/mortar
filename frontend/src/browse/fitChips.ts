/**
 * How many of the chips (by width) fit a row of the given width. When some do not, the last slot is the "+N" chip of
 * moreWidth, so the count leaves room for it. At least one chip shows when any fits at all.
 */
export function fitChips(
  widths: readonly number[],
  available: number,
  gap: number,
  moreWidth: number,
): number {
  const total = widths.reduce((sum, w, i) => sum + w + (i > 0 ? gap : 0), 0)
  if (total <= available) {
    return widths.length
  }
  let used = 0
  let shown = 0
  for (const w of widths) {
    const next = used + (shown > 0 ? gap : 0) + w
    if (next + gap + moreWidth > available) {
      break
    }
    used = next
    shown += 1
  }
  return shown
}
