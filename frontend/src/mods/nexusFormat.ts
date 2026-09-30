import type { File } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexus/models.ts'

const KB_PER_MB = 1024
const LEADING_V = /^v/i
// Go's zero time, sent for a date Nexus left out.
const FIRST_YEAR = 1970
// Nexus leaves the category out on many old files, so only these count as current.
const current = new Set(['MAIN', 'UPDATE', 'OPTIONAL', 'MISCELLANEOUS'])

const bare = (v: string) => v.trim().replace(LEADING_V, '')

export const formatSize = (kb: number, locale: string) =>
  kb < KB_PER_MB
    ? new Intl.NumberFormat(locale, { style: 'unit', unit: 'kilobyte' }).format(kb)
    : new Intl.NumberFormat(locale, {
        style: 'unit',
        unit: 'megabyte',
        maximumFractionDigits: 1,
      }).format(kb / KB_PER_MB)

export const formatCount = (n: number, locale: string) =>
  new Intl.NumberFormat(locale, { notation: 'compact', maximumFractionDigits: 1 }).format(n)

export const formatDate = (iso: string, locale: string) => {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) || d.getUTCFullYear() < FIRST_YEAR
    ? ''
    : new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(d)
}

// The installed file first, then the page's current files, newest first; old and archived files are left out.
export const currentFiles = (files: File[], installedId: number) => {
  const byNewest = (a: File, b: File) => b.uploaded.localeCompare(a.uploaded)
  const installed = files.find((f) => f.fileId === installedId)
  const others = files
    .filter((f) => f.fileId !== installedId && current.has(f.category))
    .sort(byNewest)
  return installed ? [installed, ...others] : others
}

// ponytail: numeric string order, so "1.0.0-beta" reads as newer than "1.0.0"; parse SemVer if pre-releases matter.
export const isNewer = (latest: string, installed: string) =>
  bare(latest) !== '' &&
  bare(latest).localeCompare(bare(installed), 'en', { numeric: true, sensitivity: 'base' }) > 0
