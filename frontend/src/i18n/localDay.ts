/** The local calendar day as YYYY-MM-DD: a sortable key and a file-name stamp, not text for the player's locale. */
export function localDay(d: Date): string {
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${m}-${day}`
}
