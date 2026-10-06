import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { formatAll } from './filter.ts'

const UNSAFE = /[<>:"/\\|?*]/g

const SPACES = /\s+/g
const safe = (name: string, fallback: string) =>
  name.replace(UNSAFE, '-').replace(SPACES, ' ').trim() || fallback

// logFileName names a saved log after the loader that wrote it, the profile and the day.
export function logFileName(loaderName: string, profileName: string, at: Date): string {
  const loader = safe(loaderName, 'log')
  const profile = safe(profileName, 'profile')
  const y = at.getFullYear()
  const m = String(at.getMonth() + 1).padStart(2, '0')
  const d = String(at.getDate()).padStart(2, '0')
  return `${loader}-${profile}-${y}-${m}-${d}.txt`
}

export function saveLogText(raw: string, entries: Entry[]): string {
  return raw === '' ? formatAll(entries) : raw
}
