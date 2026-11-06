import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { localDay } from '../i18n/localDay.ts'
import { formatAll } from './filter.ts'

const UNSAFE = /[<>:"/\\|?*]/g

const SPACES = /\s+/g
// safe makes a name usable in a file name on every platform, falling back when nothing is left.
const safe = (name: string, fallback: string) =>
  name.replace(UNSAFE, '-').replace(SPACES, ' ').trim() || fallback

// logFileName names a saved log after the loader that wrote it, the profile and the day.
export function logFileName(loaderName: string, profileName: string, at: Date): string {
  const loader = safe(loaderName, 'log')
  const profile = safe(profileName, 'profile')
  return `${loader}-${profile}-${localDay(at)}.txt`
}

export function saveLogText(raw: string, entries: Entry[]): string {
  return raw === '' ? formatAll(entries) : raw
}

export { safe as safeFileName }
