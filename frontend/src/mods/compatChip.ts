import { sameId } from './lookup.ts'

export function showCompatChip(status: string | undefined | null): boolean {
  const value = (status ?? '').toLowerCase()
  return value !== '' && value !== 'ok'
}

export function compatOf<T extends { id?: string; key?: string }>(
  rows: T[] | null | undefined,
  mod: { id: string; key: string },
): T | undefined {
  return (rows ?? []).find((row) => row.key === mod.key && sameId(row.id ?? '', mod.id))
}

export function compatReportChunks(
  rows: { name: string; status: string; summary: string }[],
  title: string,
): { title: string; count: number; lines: string[] }[] {
  if (rows.length === 0) {
    return []
  }
  const lines = rows.map((item) => {
    const extra = item.summary === '' ? '' : ` — ${item.summary}`
    return `${item.name}: ${item.status}${extra}`
  })
  return [{ title, count: rows.length, lines }]
}
