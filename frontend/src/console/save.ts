import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { formatAll } from './filter.ts'

const UNSAFE = /[<>:"/\\|?*]/g

export function logFileName(profileName: string, at: Date): string {
  const profile = profileName.replace(UNSAFE, '-').replace(/\s+/g, ' ').trim() || 'profile'
  const y = at.getFullYear()
  const m = String(at.getMonth() + 1).padStart(2, '0')
  const d = String(at.getDate()).padStart(2, '0')
  return `SMAPI-${profile}-${y}-${m}-${d}.txt`
}

export function saveLogText(raw: string, entries: Entry[]): string {
  return raw === '' ? formatAll(entries) : raw
}
