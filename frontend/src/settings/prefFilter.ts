export function prefMatches(query: string, label: string, description = ''): boolean {
  const q = query.trim().toLowerCase()
  if (!q) {
    return true
  }
  return label.toLowerCase().includes(q) || description.toLowerCase().includes(q)
}

export function sectionVisible(
  query: string,
  rows: { label: string; description?: string }[],
): boolean {
  return rows.some((row) => prefMatches(query, row.label, row.description ?? ''))
}
