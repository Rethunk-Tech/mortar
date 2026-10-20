// The demo's settings for the visit: the recorded ones, changed in memory as the visitor changes them.
import { isRecord } from './replay.ts'
import { emit } from './run.ts'

type Block = Record<string, unknown>

const lower = (field: string) => field.charAt(0).toLowerCase() + field.slice(1)

export class Settings {
  private current: Block | undefined

  get(recorded: unknown): Block {
    this.current ??= isRecord(recorded) ? structuredClone(recorded) : {}
    return this.current
  }

  private changed() {
    emit('settings:changed', this.current)
  }

  private block(game: string): Block | undefined {
    if (!this.current) {
      return undefined
    }
    if (game === '') {
      return this.current
    }
    const games = isRecord(this.current.games) ? this.current.games : {}
    this.current.games = games
    const block = isRecord(games[game]) ? games[game] : {}
    games[game] = block
    return block
  }

  /** A setting by its key, typed as its spec declares. */
  setByKey(key: string, value: string, game: string, specs: unknown) {
    const spec = Array.isArray(specs) ? specs.find((s) => isRecord(s) && s.key === key) : undefined
    const type = isRecord(spec) ? spec.type : 'string'
    const block = this.block(isRecord(spec) && spec.scope === 'game' ? game : '')
    if (!block) {
      return
    }
    if (type === 'bool') {
      block[key] = value === 'true'
    } else if (type === 'int') {
      block[key] = Number(value)
    } else {
      block[key] = value
    }
    this.changed()
  }

  /** A setter named for its field (SetAccent writes accent), Mortar-wide or for a game first; false when none. */
  set(field: string, args: unknown[]): boolean {
    const name = lower(field)
    const value = args.at(-1)
    const game = args.length > 1 && typeof args[0] === 'string' ? args[0] : ''
    const block = this.block(game)
    if (!block || (game === '' && !(name in block))) {
      return false
    }
    block[name] = value
    this.changed()
    return true
  }
}
