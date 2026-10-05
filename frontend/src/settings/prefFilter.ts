// Words people type that the labels do not say. Each trigger also searches for the listed label words; a
// trigger matches when the query is the trigger or a start of it (three letters or more, so "pro" is not "proxy").
const SYNONYMS: Record<string, string[]> = {
  dark: ['theme'],
  light: ['theme'],
  mode: ['theme'],
  night: ['theme'],
  appearance: ['theme', 'density'],
  proxy: ['local network', 'lan port', 'automatic port'],
  network: ['local network', 'lan port'],
  firewall: ['local network', 'lan port'],
  'api key': ['account', 'api key'],
  apikey: ['account', 'api key'],
  token: ['account', 'api key'],
  login: ['account', 'sign in'],
  password: ['account', 'sign in'],
  trash: ['recently deleted', 'storage'],
  recycle: ['recently deleted', 'storage'],
  cleanup: ['clean up', 'storage'],
  disk: ['storage', 'usage'],
  space: ['storage', 'usage'],
  cache: ['clear cache', 'storage'],
  startup: ['login', 'tray'],
  autostart: ['login', 'tray'],
}

const MIN_PREFIX = 3

function queryTerms(query: string): string[] {
  const q = query.trim().toLowerCase()
  const extra = Object.entries(SYNONYMS).flatMap(([trigger, terms]) =>
    q === trigger || (q.length >= MIN_PREFIX && trigger.startsWith(q)) ? terms : [],
  )
  return [q, ...extra]
}

export function prefMatches(query: string, label: string, description = ''): boolean {
  if (!query.trim()) {
    return true
  }
  const haystack = `${label}\n${description}`.toLowerCase()
  return queryTerms(query).some((term) => haystack.includes(term))
}

export function sectionVisible(
  query: string,
  rows: { label: string; description?: string }[],
): boolean {
  return rows.some((row) => prefMatches(query, row.label, row.description ?? ''))
}
