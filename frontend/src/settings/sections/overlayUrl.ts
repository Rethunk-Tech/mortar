import { sentence } from '../../i18n/sentence.ts'

const STARDEW_FIELDS = [
  'location',
  'player',
  'season',
  'day',
  'year',
  'time',
  'date',
  'money',
  'weather',
  'health',
  'stamina',
  'skill.farming',
  'skill.fishing',
  'skill.foraging',
  'skill.mining',
  'skill.combat',
  'skill.luck',
] as const

// The bridge names its game in /state; a snapshot without one (the SMAPI bridge's) is Stardew Valley's.
const GAME_FIELDS: Record<string, readonly string[]> = {
  'lethal-company': ['player', 'moon', 'crew', 'day', 'quota', 'daysLeft', 'credits'],
  valheim: ['player', 'biome', 'day', 'bosses', 'bossList'],
}

function overlayFields(game: string): readonly string[] {
  return GAME_FIELDS[game] ?? STARDEW_FIELDS
}

const BOSS_COUNT = 7
const TIME_SCALE = 100
const HALF_DAY_HOURS = 12

function overlayPageUrl(base: string, field?: string, labels = false): string {
  const u = new URL(base)
  if (field) {
    u.searchParams.set('field', field)
  } else {
    u.searchParams.delete('field')
  }
  if (labels) {
    u.searchParams.set('label', '1')
  } else {
    u.searchParams.delete('label')
  }
  return u.toString()
}

type OverlayPreview =
  | { kind: 'unreachable' }
  | { kind: 'notInGame' }
  | { kind: 'value'; text: string }

function seasonLabel(season: unknown): string {
  const s = String(season ?? '')
  return sentence(s)
}

function formatTime(timeOfDay: unknown): string {
  const n = Number(timeOfDay)
  if (!Number.isFinite(n)) {
    return ''
  }
  let hours = Math.floor(n / TIME_SCALE)
  const minutes = String(n % TIME_SCALE).padStart(2, '0')
  const period = hours >= HALF_DAY_HOURS ? 'PM' : 'AM'
  hours %= HALF_DAY_HOURS
  if (hours === 0) {
    hours = HALF_DAY_HOURS
  }
  return `${hours}:${minutes} ${period}`
}

function skillLevel(body: Record<string, unknown>, skill: string): string {
  const { skills } = body
  if (skills === null || typeof skills !== 'object') {
    return ''
  }
  const level = (skills as Record<string, unknown>)[skill]
  if (level === undefined || level === null) {
    return ''
  }
  return String(level)
}

function missing(value: unknown): boolean {
  return value === undefined || value === null
}

function fraction(a: unknown, b: unknown, suffix = ''): string {
  return missing(a) || missing(b) ? '' : `${a}/${b}${suffix}`
}

function gameField(body: Record<string, unknown>, field: string): string {
  const text = (v: unknown) => (missing(v) ? '' : String(v))
  switch (field) {
    case 'player':
      return text(body.playerName)
    case 'crew':
      return fraction(body.crewAlive, body.crewTotal, ' alive')
    case 'quota':
      return fraction(body.quotaProgress, body.quota)
    case 'credits':
      return missing(body.credits) ? '' : `\u25A0${body.credits}`
    case 'bosses':
      return missing(body.bossCount) ? '' : `${body.bossCount}/${BOSS_COUNT} defeated`
    case 'bossList':
      return Array.isArray(body.bossesDefeated) ? body.bossesDefeated.join(', ') : ''
    default:
      return text(body[field])
  }
}

function formatField(body: Record<string, unknown>, field: string): string {
  if (typeof body.game === 'string' && body.game in GAME_FIELDS) {
    return gameField(body, field)
  }
  switch (field) {
    case 'location':
      return String(body.location ?? '')
    case 'player':
      return String(body.playerName ?? '')
    case 'season':
      return seasonLabel(body.season)
    case 'day':
      return missing(body.day) ? '' : String(body.day)
    case 'year':
      return missing(body.year) ? '' : String(body.year)
    case 'time':
      return formatTime(body.timeOfDay)
    case 'date':
      return `${seasonLabel(body.season)} ${body.day}, Year ${body.year}`
    case 'money':
      return missing(body.money) ? '' : `${body.money}g`
    case 'weather':
      return String(body.weather ?? '')
    case 'health':
      return `${body.health}/${body.maxHealth}`
    case 'stamina':
      return `${Math.round(Number(body.stamina))}/${body.maxStamina}`
    default:
      if (field.startsWith('skill.')) {
        return skillLevel(body, field.slice('skill.'.length))
      }
      return ''
  }
}

function overlayPreview(
  fetched: { ok: true; body: Record<string, unknown> } | { ok: false } | null,
  field?: string,
): OverlayPreview {
  if (fetched === null || !fetched.ok) {
    return { kind: 'unreachable' }
  }
  const { body } = fetched
  if (body.inGame !== true) {
    return { kind: 'notInGame' }
  }
  if (!field) {
    return {
      kind: 'value',
      text: overlayFields(String(body.game ?? ''))
        .map((f) => formatField(body, f))
        .join(' · '),
    }
  }
  return { kind: 'value', text: formatField(body, field) }
}

const OVERLAY_EXAMPLE_CSS = `body { background: transparent; margin: 0; padding: 0; }
#player, .player { font-weight: 700; color: #fff; }
#money, .money { color: #ffe566; font-size: 28px; }
#skill-mining, .skill.mining { color: #9ad1ff; }
#time, .time { font-variant-numeric: tabular-nums; }
`

export {
  OVERLAY_EXAMPLE_CSS,
  type OverlayPreview,
  overlayFields,
  overlayPageUrl,
  overlayPreview,
  STARDEW_FIELDS,
}
