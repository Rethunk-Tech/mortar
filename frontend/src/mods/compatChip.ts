import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
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
    const extra = item.summary === '' ? '' : ` · ${item.summary}`
    return `${item.name}: ${item.status}${extra}`
  })
  return [{ title, count: rows.length, lines }]
}

const STATUS_LABELS = {
  optional: msg`Optional`,
  unofficial: msg`Unofficial`,
  broken: msg`Broken`,
  obsolete: msg`Obsolete`,
  abandoned: msg`Abandoned`,
  superseded: msg`Superseded`,
  notrecommended: msg`Not recommended`,
} as const

/** A compatibility status as the player reads it; a status Mortar has no word for shows as the list wrote it. */
export function compatStatusLabel(status: string | undefined | null): string {
  const raw = status ?? ''
  const key = raw.toLowerCase()
  return key in STATUS_LABELS ? i18n._(STATUS_LABELS[key as keyof typeof STATUS_LABELS]) : raw
}
