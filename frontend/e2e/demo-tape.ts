// Captures the demo recorder's backend traffic by phase, and assembles it into the recording the demo replays.

import { readFileSync } from 'node:fs'
import type { Page, Request } from '@playwright/test'
import {
  type Calls,
  callKey,
  callOf,
  isProfile,
  isRecord,
  modState,
  type Recording,
  type Reply,
  type TimedCall,
  type TimedEvent,
} from '../src/demo/replay.ts'

// A bound method's id, read from its generated binding (frontend/bindings/…/internal/<folder>/service.ts).
function methodId(name: string): number {
  const [folder, method] = name.split('.Service.')
  const source = readFileSync(
    new URL(
      `../bindings/github.com/Rethunk-Tech/mortar/internal/${folder}/service.ts`,
      import.meta.url,
    ),
    'utf8',
  )
  const found = new RegExp(`export function ${method}\\([^]*?\\$Call\\.ByID\\((\\d+)`).exec(source)
  return Number(found?.[1] ?? 0)
}

export type Phase = 'off' | 'base' | 'toggle' | 'run' | 'after' | 'blank'

interface Taped {
  phase: Phase
  at: number
  key: string
  reply: Reply
}

const GAME_LIST = methodId('game.Service.List')
const SEARCH = methodId('browse.Service.Search')
// A source's first page is all the demo shows; it asks for no more.
const SEARCH_ITEMS = 24
const STARTS = new Set(
  ['Start', 'StartPreset', 'StartVanilla'].map((m) => methodId(`launchsvc.Service.${m}`)),
)
const STOP = methodId('launchsvc.Service.Stop')
// Art and icons stay with their sites: Mortar never re-hosts them, and the demo fetches nothing but its own files.
const PICTURE_KEY =
  /^(picture|icon|image|art|cover|thumb|avatar|logo|banner)|(Picture|Icon|Image|Art|Cover|Thumb|Avatar|Logo|Banner)/

// The backend's own art route serves Steam's art, which the demo leaves out too.
const STEAM_ART = '/steam-art/'

function blankPictures(v: unknown): unknown {
  if (Array.isArray(v)) {
    return v.filter((x) => !(typeof x === 'string' && x.startsWith(STEAM_ART))).map(blankPictures)
  }
  if (!isRecord(v)) {
    return v
  }
  return Object.fromEntries(
    Object.entries(v).map(([k, x]) => [
      k,
      typeof x === 'string' && PICTURE_KEY.test(k) && /^(https?:|\/)/.test(x)
        ? ''
        : blankPictures(x),
    ]),
  )
}

// The catalog's game list keeps the installed games.
const installedGames = (games: unknown): unknown =>
  Array.isArray(games) ? games.filter((g) => isRecord(g) && g.installed === true) : games

async function replyOf(request: Request): Promise<Reply | undefined> {
  const response = await request.response()
  if (!response) {
    return undefined
  }
  const status = response.status()
  const text = await response.text()
  if (!(response.headers()['content-type'] ?? '').includes('application/json')) {
    return { status, text }
  }
  return { status, json: blankPictures(JSON.parse(text)) }
}

/** Every runtime call and backend event the page sees, stamped with the phase it fell in. */
export class Tape {
  phase: Phase = 'off'
  private readonly calls: Promise<Taped | undefined>[] = []
  private readonly events: (TimedEvent & { phase: Phase })[] = []
  private readonly started = new Map<Request, { phase: Phase; at: number }>()

  constructor(page: Page) {
    page.on('request', (request) => {
      if (new URL(request.url()).pathname === '/wails/runtime') {
        this.started.set(request, { phase: this.phase, at: Date.now() })
      }
    })
    page.on('requestfinished', (request) => {
      const start = this.started.get(request)
      const body = request.postData()
      if (start && body) {
        this.calls.push(
          replyOf(request)
            .then((reply) => reply && { ...start, key: callKey(body), reply })
            // A call still answering when the browser closes was not part of any view.
            .catch(() => undefined),
        )
      }
    })
    page.on('websocket', (socket) => {
      socket.on('framereceived', ({ payload }) => {
        const event: unknown = typeof payload === 'string' ? JSON.parse(payload) : undefined
        if (this.phase !== 'off' && isRecord(event) && typeof event.name === 'string') {
          this.events.push({
            phase: this.phase,
            at: Date.now(),
            name: event.name,
            data: blankPictures(event.data),
          })
        }
      })
    })
  }

  async assemble(crewId: string, blankId: string): Promise<Recording> {
    const taped = (await Promise.all(this.calls)).filter((t): t is Taped => t !== undefined)
    const of = (phase: Phase) => taped.filter((t) => t.phase === phase)
    const run = of('run')
    const start = run.find((t) => STARTS.has(callOf(t.key).method))
    const t0 = start?.at ?? 0
    const stop = run.find((t) => STOP === callOf(t.key).method)
    const events = this.events
      .filter((e) => e.phase === 'run' && e.at >= t0)
      .map(({ at, name, data }) => ({ at: at - t0, name, data }))
    const blank = firstOf(of('blank').filter((t) => t.key.includes(blankId)))
    return {
      calls: firstOf([...of('base'), ...run.filter((t) => t.at < t0)]),
      states: stateOf(of('toggle'), crewId),
      blank: { id: blankId, profile: listed(taped, blankId) ?? null, calls: blank },
      run: {
        wall: t0,
        start: start?.key ?? '',
        stop: stop?.key ?? '',
        stopAt: (stop?.at ?? t0) - t0,
        endAt: Math.max(0, ...events.map((e) => e.at)),
        events,
        calls: run
          .filter((t) => t.at >= t0)
          .map((t): TimedCall => ({ at: t.at - t0, key: t.key, reply: t.reply })),
      },
      after: firstOf(of('after')),
    }
  }
}

function trimmed(key: string, reply: Reply): Reply {
  const { method } = callOf(key)
  if (!('json' in reply)) {
    return reply
  }
  if (method === GAME_LIST) {
    return { ...reply, json: installedGames(reply.json) }
  }
  const { json } = reply
  if (method === SEARCH && isRecord(json) && Array.isArray(json.items)) {
    return { ...reply, json: { ...json, pages: 1, items: json.items.slice(0, SEARCH_ITEMS) } }
  }
  return reply
}

// The first answer is the state a view opens on; later repeats are the same call refreshed.
function firstOf(taped: Taped[]): Calls {
  const calls: Calls = {}
  for (const t of taped) {
    if (!(t.key in calls)) {
      calls[t.key] = trimmed(t.key, t.reply)
    }
  }
  return calls
}

// The profile with this id as the first answer that carries it has it.
function listed(taped: Taped[], id: string): unknown {
  const within = (v: unknown): unknown => {
    if (Array.isArray(v)) {
      return v.map(within).find((x) => x !== undefined)
    }
    if (!isRecord(v)) {
      return undefined
    }
    return isProfile(v) && v.id === id
      ? v
      : Object.values(v)
          .map(within)
          .find((x) => x !== undefined)
  }
  for (const t of taped) {
    const found = 'json' in t.reply ? within(t.reply.json) : undefined
    if (found) {
      return found
    }
  }
  return undefined
}

// The toggled views, keyed by the mod state of the profile the toggle changed.
function stateOf(taped: Taped[], id: string): Recording['states'] {
  const changed = listed(taped, id)
  return changed ? { [modState(changed)]: firstOf(taped) } : {}
}
