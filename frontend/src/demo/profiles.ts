// The demo's profiles in memory for the visit: created, copied, renamed, deleted and their mods switched, with
// every recorded answer that carries a profile shown as it now stands.
import { isProfile, isRecord, modState, type Recording } from './replay.ts'

type Profile = Record<string, unknown>

const ID_BYTES = 8
const HEX = 16
const BYTE_DIGITS = 2

const newId = () =>
  Array.from(crypto.getRandomValues(new Uint8Array(ID_BYTES)), (b) =>
    b.toString(HEX).padStart(BYTE_DIGITS, '0'),
  ).join('')

const clone = (p: unknown): Profile => (isRecord(p) ? structuredClone(p) : {})

export class Profiles {
  private readonly lists = new Map<string, Profile[]>()
  /** A profile made in the visit, to the recorded one whose answers it reads. */
  private readonly sources = new Map<string, string>()
  private readonly blank: Recording['blank']
  last = ''

  constructor(recording: Recording) {
    this.blank = recording.blank
  }

  /** The game's profiles, from the first recorded list until the visit changes them. */
  list(game: string, recorded: unknown): Profile[] {
    let list = this.lists.get(game)
    if (!list) {
      list = Array.isArray(recorded) ? recorded.filter(isRecord).map(clone) : []
      this.lists.set(game, list)
    }
    return list
  }

  find(id: string): Profile | undefined {
    for (const list of this.lists.values()) {
      const found = list.find((p) => p.id === id)
      if (found) {
        return found
      }
    }
    return undefined
  }

  /** The recorded profile a call about this one is answered from, and the id to answer with. */
  source(id: string): string | undefined {
    return this.sources.get(id)
  }

  isBlank = (id: string) => this.blank.id !== '' && id === this.blank.id

  add(game: string, from: unknown, name: string, source: string): Profile {
    const p = clone(from)
    const id = newId()
    const now = new Date().toISOString()
    Object.assign(p, { id, name, created: now, updated: now })
    this.sources.set(id, this.sources.get(source) ?? source)
    this.lists.get(game)?.push(p)
    return p
  }

  create(game: string, name: string): Profile {
    return this.add(game, this.blank.profile, name, this.blank.id)
  }

  remove(game: string, id: string) {
    const list = this.lists.get(game)
    if (list) {
      this.lists.set(
        game,
        list.filter((p) => p.id !== id),
      )
    }
  }

  /** Switches one of a profile's mods, as the profile's entry records it: the ids it holds switched off. */
  setEnabled(id: string, key: string, mod: string, on: boolean): Profile | undefined {
    const p = this.find(id)
    const entries = Array.isArray(p?.entries) ? p.entries.filter(isRecord) : []
    const entry = entries.find((e) => e.key === key)
    if (entry) {
      const off = new Set(Array.isArray(entry.disabled) ? entry.disabled.map(String) : [])
      if (on) {
        off.delete(mod)
      } else {
        off.add(mod)
      }
      entry.disabled = [...off]
    }
    return p
  }

  state(id: string): string {
    return modState(this.find(id))
  }

  /** A recorded answer with every profile it carries as the visit has left it, and each mod's switch to match. */
  patch(value: unknown): unknown {
    if (Array.isArray(value)) {
      return value.map((v) => this.patch(v))
    }
    if (!isRecord(value)) {
      return value
    }
    const p = isProfile(value) ? this.find(String(value.id)) : undefined
    if (p) {
      return p
    }
    const out: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(value)) {
      out[k] = this.patch(v)
    }
    return out
  }

  /** The mod list of a profile with each mod switched as its entry now has it. */
  mods(id: string, list: unknown): unknown {
    const p = this.find(id)
    const entries = Array.isArray(p?.entries) ? p.entries.filter(isRecord) : []
    if (!Array.isArray(list)) {
      return list
    }
    return list.map((m) => {
      const entry = isRecord(m) ? entries.find((e) => e.key === m.key) : undefined
      if (!(isRecord(m) && entry)) {
        return m
      }
      const off = Array.isArray(entry.disabled) ? entry.disabled.map(String) : []
      return { ...m, enabled: !off.includes(String(m.id)) }
    })
  }
}
