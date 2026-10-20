// One recorded play of the demo profile, replayed on Play: the backend's events at their recorded times, and the
// answers the views got at that point of the run. Once it ends, the views read the state the run left.
import { shifted } from './clock.ts'
import type { Recording, Reply, TimedEvent } from './replay.ts'

type Phase = 'before' | 'running' | 'ended'

interface WailsWindow {
  _wails?: { dispatchWailsEvent?: (event: { name: string; data: unknown }) => void }
}

export function emit(name: string, data: unknown) {
  ;(globalThis as unknown as WailsWindow)._wails?.dispatchWailsEvent?.({ name, data })
}

export class Run {
  private phase: Phase = 'before'
  private startedAt = 0
  private playedAt = 0
  private readonly loadedAt = Date.now()
  /** The profile played, whose id the recorded run's events carry in place of the recorded one. */
  private as = ''
  private timers: ReturnType<typeof setTimeout>[] = []
  private readonly run: Recording['run']
  private readonly after: Recording['after']

  constructor(recording: Recording) {
    this.run = recording.run
    this.after = recording.after
  }

  /** Whether the recording holds a run to play. */
  ready = () => this.run.start !== ''

  /** How far the recorded times move: the recorded start to Play's, or to the page's load before any Play. */
  delta = () => (this.playedAt || this.loadedAt) - this.run.wall

  /** Answers or event data with the recorded times moved to the visit. */
  now = (value: unknown) => shifted(value, this.delta(), this.run.wall)

  start(profile: string, recorded: string): Reply | undefined {
    this.phase = 'running'
    this.startedAt = Date.now()
    this.playedAt = this.startedAt
    this.as = profile === recorded ? '' : profile
    this.recorded = recorded
    this.schedule(this.run.events, 0)
    return this.at(this.run.start)
  }

  /** Stop jumps to where the recorded run was stopped, and plays on from there. */
  stop(key: string): Reply | undefined {
    if (this.phase !== 'running' || key !== this.run.stop) {
      return undefined
    }
    if (this.elapsed() < this.run.stopAt) {
      this.startedAt = Date.now() - this.run.stopAt
      this.schedule(
        this.run.events.filter((e) => e.at >= this.run.stopAt),
        this.run.stopAt,
      )
    }
    return this.at(key)
  }

  /** The answer the run recorded for a call at this point of it, else the answer once it ended. */
  answer(key: string): Reply | undefined {
    if (this.phase === 'running') {
      return this.at(key)
    }
    return this.phase === 'ended' ? this.after[key] : undefined
  }

  private recorded = ''

  private data(data: unknown): unknown {
    const moved = this.now(data)
    return this.as === '' || moved === null || typeof moved !== 'object'
      ? moved
      : JSON.parse(JSON.stringify(moved).replaceAll(this.recorded, this.as))
  }

  private readonly elapsed = () => Date.now() - this.startedAt

  private at(key: string): Reply | undefined {
    const now = this.elapsed()
    let found: Reply | undefined
    for (const c of this.run.calls) {
      if (c.key === key && (found === undefined || c.at <= now)) {
        found = c.reply
      }
    }
    return found
  }

  private schedule(events: TimedEvent[], from: number) {
    for (const timer of this.timers) {
      clearTimeout(timer)
    }
    this.timers = events.map((e) => setTimeout(() => emit(e.name, this.data(e.data)), e.at - from))
    this.timers.push(
      setTimeout(
        () => {
          this.phase = 'ended'
        },
        this.run.endAt - from + 1,
      ),
    )
  }
}
