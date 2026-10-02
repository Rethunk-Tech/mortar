import type { File } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexus/models.ts'

const KB_PER_MB = 1024
// Go's zero time, sent for a date Nexus left out.
// Nexus leaves the category out on many old files, so only these count as current.
const current = new Set(['MAIN', 'UPDATE', 'OPTIONAL', 'MISCELLANEOUS'])

const VERSION =
  /^v?(\d+)\.(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$/

const ATOI = /^[+-]?\d+$/

const CHANGELOG_CAP = 5

const atoi = (s: string) => (ATOI.test(s) ? Number(s) : null)

const sign = (n: number) => {
  if (n < 0) {
    return -1
  }
  if (n > 0) {
    return 1
  }
  return 0
}

const comparePre = (a: string, b: string) => {
  const na = atoi(a)
  const nb = atoi(b)
  if (na !== null && nb !== null) {
    return sign(na - nb)
  }
  if (na !== null) {
    return -1
  }
  if (nb !== null) {
    return 1
  }
  return sign(a.toLowerCase().localeCompare(b.toLowerCase()))
}

const parseVersion = (s: string) => {
  const m = s.trim().match(VERSION)
  if (!m) {
    return null
  }
  const [, major, minor, patch, revision, pre] = m
  return {
    nums: [
      Number(major),
      Number(minor),
      patch ? Number(patch) : 0,
      revision ? Number(revision) : 0,
    ],
    pre: pre ? pre.split('.') : [],
  }
}

const compareVersions = (a: string, b: string) => {
  const x = parseVersion(a)
  const y = parseVersion(b)
  if (!(x && y)) {
    return null
  }
  for (let i = 0; i < x.nums.length; i++) {
    const xn = x.nums[i] ?? 0
    const yn = y.nums[i] ?? 0
    if (xn !== yn) {
      return sign(xn - yn)
    }
  }
  if (x.pre.length === 0 && y.pre.length === 0) {
    return 0
  }
  if (x.pre.length === 0) {
    return 1
  }
  if (y.pre.length === 0) {
    return -1
  }
  const n = Math.min(x.pre.length, y.pre.length)
  for (let i = 0; i < n; i++) {
    const left = x.pre[i]
    const right = y.pre[i]
    if (left === undefined || right === undefined) {
      break
    }
    const c = comparePre(left, right)
    if (c !== 0) {
      return c
    }
  }
  return sign(x.pre.length - y.pre.length)
}

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

// The installed file first, then the page's current files, newest first; old and archived files are left out.
export const currentFiles = (files: File[], installedId: number) => {
  const byNewest = (a: File, b: File) => b.uploaded.localeCompare(a.uploaded)
  const installed = files.find((f) => f.fileId === installedId)
  const others = files
    .filter((f) => f.fileId !== installedId && current.has(f.category))
    .sort(byNewest)
  return installed ? [installed, ...others] : others
}

export const isNewer = (latest: string, installed: string) => {
  const cmp = compareVersions(latest, installed)
  return cmp !== null && cmp > 0
}

export const recentChangelogs = <T>(logs: T[]) => logs.slice(0, CHANGELOG_CAP)
