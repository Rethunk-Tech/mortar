import { msg } from '@lingui/core/macro'
import { useEffect, useRef } from 'react'
import { State } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import {
  History,
  MarkKnownGood,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useLaunch } from './store.ts'

const KNOWN_GOOD = 'good'

async function markKnownGood(game: string, profile: string) {
  try {
    const marked = await MarkKnownGood(game, profile)
    useToasts.getState().push({
      kind: 'success',
      title: i18n._(msg`Marked known good`),
      changes: [marked.id],
    })
  } catch (e) {
    reportUnexpected(e)
  }
}

// After a clean run, offers to mark the profile's mods known good, unless they are unchanged since the last mark:
// asking after every run with the same mods is noise.
export function KnownGoodOffer() {
  const status = useLaunch((s) => s.status)
  const prev = useRef(status)
  useEffect(() => {
    const was = prev.current
    prev.current = status
    if (
      status?.state !== State.Idle ||
      was?.state !== State.Running ||
      status.game === '' ||
      status.profile === ''
    ) {
      return
    }
    const { game, profile } = status
    Promise.all([Runs(game, profile), History(game, profile)])
      .then(([runs, history]) => {
        const run = runs?.[0]
        const clean = run && run.outcome === 'ran' && (run.errors ?? 0) === 0
        if (!clean || history?.[0]?.kind === KNOWN_GOOD) {
          return
        }
        useToasts.getState().push({
          kind: 'success',
          title: i18n._(msg`The game started without errors`),
          body: i18n._(
            msg`Mark these mods known good so History can restore them if a later change breaks the game.`,
          ),
          action: {
            label: i18n._(msg`Mark known good`),
            profileId: profile,
            run: () => markKnownGood(game, profile),
          },
        })
      })
      .catch(() => undefined)
  }, [status])
  return null
}
