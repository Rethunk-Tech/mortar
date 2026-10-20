// The demo's backend: recorded answers, with the visit's own changes kept in memory on top of them.
import { emptyOf, idOf, methodOf } from './methods.ts'
import { outsideDemo } from './notice.ts'
import { Profiles } from './profiles.ts'
import { callOf, isRecord, keyOf, type Recording, type Reply } from './replay.ts'
import { emit, Run } from './run.ts'
import { Settings } from './settings.ts'

// game, profile id and the value.
const PROFILE_SETTER_ARGS = 3

const PLAY = new Set(['launchsvc.Start', 'launchsvc.StartPreset', 'launchsvc.StartVanilla'])

type Handler = (args: unknown[], recorded: () => unknown) => unknown

const str = (v: unknown) => (typeof v === 'string' ? v : '')

export class Backend {
  private readonly recording: Recording
  private readonly profiles: Profiles
  private readonly settings: Settings
  private readonly run: Run
  private readonly demoProfile: string
  private readonly handlers: Record<string, Handler>

  constructor(recording: Recording) {
    this.recording = recording
    this.profiles = new Profiles(recording)
    this.settings = new Settings()
    this.run = new Run(recording)
    this.demoProfile = str(callOf(recording.run.start).args[1])
    this.handlers = this.makeHandlers()
  }

  answer(key: string): Reply {
    const reply = this.reply(key)
    return 'json' in reply ? { ...reply, json: this.run.now(reply.json) } : reply
  }

  private reply(key: string): Reply {
    const { method, args } = callOf(key)
    const m = methodOf(method)
    const name = m?.name ?? ''
    for (const a of args) {
      if (typeof a === 'string' && this.profiles.find(a)) {
        this.profiles.last = a
      }
    }
    const handler = this.handlers[name]
    if (handler) {
      return ok(handler(args, () => this.json(key)))
    }
    if (
      name.startsWith('settings.Set') &&
      this.settings.set(name.slice('settings.Set'.length), args)
    ) {
      return ok(null)
    }
    if (PLAY.has(name) && this.run.ready()) {
      return this.run.start(str(args[1]), this.demoProfile) ?? ok(null)
    }
    const stopped = name === 'launchsvc.Stop' ? this.run.stop(key) : undefined
    const reply = stopped ?? this.recorded(key, method, args)
    if (reply) {
      return reply
    }
    if (outsideDemo(name)) {
      return ok(emptyOf(m?.result ?? 'void'))
    }
    return ok(this.fallback(name, m?.result ?? 'void', args))
  }

  // A profile made in the visit is answered as the recorded profile it came from, under its own id.
  private recorded(key: string, method: number, args: unknown[]): Reply | undefined {
    const made = args.map((a) => (typeof a === 'string' ? this.profiles.source(a) : undefined))
    const i = made.findIndex((s) => s !== undefined)
    const source = made[i]
    if (source === undefined) {
      return this.finish(this.lookup(key), key, args)
    }
    const asSource = keyOf(
      method,
      args.map((a, j) => (j === i ? source : a)),
    )
    const reply = this.lookup(asSource) ?? this.recording.blank.calls[asSource]
    if (!(reply && 'json' in reply)) {
      return reply
    }
    const own = JSON.parse(JSON.stringify(reply.json).replaceAll(source, str(args[i])))
    return this.finish({ ...reply, json: own }, key, args)
  }

  private lookup(key: string): Reply | undefined {
    const state = this.recording.states[this.profiles.state(this.profiles.last)]
    // A view in another mod state reads that state; one the run changed reads the run.
    return state?.[key] ?? this.run.answer(key) ?? this.recording.calls[key]
  }

  // Every profile in an answer as the visit has left it.
  private finish(reply: Reply | undefined, key: string, args: unknown[]): Reply | undefined {
    if (!(reply && 'json' in reply)) {
      return reply
    }
    const json = this.profiles.patch(reply.json)
    const name = methodOf(callOf(key).method)?.name
    return {
      ...reply,
      json: name === 'profile.Mods' ? this.profiles.mods(str(args[1]), json) : json,
    }
  }

  private json(key: string): unknown {
    const reply = this.recorded(key, callOf(key).method, callOf(key).args)
    return reply && 'json' in reply ? reply.json : null
  }

  // A profile setter of one value (SetNotes, SetFavourite) keeps it under its field; any other change the demo does
  // not model leaves the profile as it was, which is what the views read back.
  private fallback(name: string, result: string, args: unknown[]): unknown {
    if (result === '$models.Profile' || result === 'profile$0.Profile') {
      const p = this.profiles.find(str(args[1]))
      const field = name.startsWith('profile.Set') ? name.slice('profile.Set'.length) : ''
      if (p && field !== '' && args.length === PROFILE_SETTER_ARGS) {
        p[field.charAt(0).toLowerCase() + field.slice(1)] = args[2]
      }
      return p ?? {}
    }
    return emptyOf(result)
  }

  private changed(game: string) {
    emit('profiles:changed', game)
  }

  private makeHandlers(): Record<string, Handler> {
    const p = this.profiles
    const enabled = (profile: unknown) => ({ profile, alsoEnabled: [] })
    return {
      'profile.List': ([game], recorded) => p.list(str(game), recorded()),
      'profile.Create': ([game, name]) => p.create(str(game), str(name)),
      'profile.Duplicate': ([game, id]) => {
        const from = p.find(str(id))
        return p.add(str(game), from, `${str(from?.name)} (copy)`, str(id))
      },
      'profile.Rename': ([, id, name]) => Object.assign(p.find(str(id)) ?? {}, { name: str(name) }),
      'profile.SetAppearance': ([, id, color, icon, description]) =>
        Object.assign(p.find(str(id)) ?? {}, { color, icon, description }),
      'profile.Delete': ([game, id]) => {
        p.remove(str(game), str(id))
        this.changed(str(game))
        return null
      },
      'profile.SetModEnabled': ([, id, key, mod, on]) =>
        enabled(p.setEnabled(str(id), str(key), str(mod), on === true)),
      'profile.SetModsEnabled': ([, id, refs, on]) => {
        for (const r of Array.isArray(refs) ? refs.filter(isRecord) : []) {
          p.setEnabled(str(id), str(r.key), str(r.id), on === true)
        }
        return enabled(p.find(str(id)))
      },
      'templates.NewProfileFromTemplate': ([game, , name]) => ({
        profile: p.add(str(game), p.find(this.demoProfile), str(name), this.demoProfile),
        missing: [],
        missingMods: [],
      }),
      'settings.Get': (_, recorded) => this.settings.get(recorded()),
      'settings.SetByKey': ([key, value, game]) => {
        this.settings.setByKey(
          str(key),
          str(value),
          str(game),
          this.json(keyOf(idOf('settings.PrefSpecs'), [])),
        )
        return null
      },
    }
  }
}

const ok = (json: unknown): Reply => ({ status: 200, json })
