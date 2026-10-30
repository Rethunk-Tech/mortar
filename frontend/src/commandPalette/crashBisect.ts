import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { useLoaded } from '@rethunk/react-load-hooks'
import { Start as StartBisect } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bisect/service.ts'
import type { Run } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { type BisectBlock, bisectBlock } from '../console/bisectBlock.ts'
import { i18n } from '../i18n/index.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useCommandPalette } from './store.ts'

// Built on call, not at import: Lingui macros only run inside compiled code.
function blockText(block: BisectBlock, lang: I18n): string {
  switch (block) {
    case 'no-profile':
      return lang._(msg`Open profile first.`)
    case 'no-runs':
      return lang._(msg`Start the game once so Mortar can see the crash`)
    case 'ended-normally':
      return lang._(msg`No crash to investigate: the last run ended normally`)
    default:
      return lang._(msg`Mortar already named the likely mod for the last crash`)
  }
}

async function blockOf(game: string | undefined, profile: string | undefined) {
  if (!(game && profile)) {
    return 'no-profile'
  }
  return bisectBlock(await Runs(game, profile))
}

/** Starts the crash check on the open profile, once the player confirmed it. */
export async function startCrashBisect(): Promise<void> {
  const { game, openId } = useProfiles.getState()
  const id = await StartBisect(game?.id ?? '', openId ?? '')
  useCommandPalette.setState({
    confirmingBisect: false,
    bisect: { id, game: game?.id ?? '', profile: openId ?? '' },
  })
}

/** Why the open profile has no crash to find, in words; null when there is one. */
export function useBisectBlockText(): string | null {
  const { i18n: lang } = useLingui()
  const game = useProfiles((s) => s.game?.id)
  const profile = useProfiles((s) => s.openId)
  const crashId = useLaunch((s) => s.crash?.runId ?? '')
  const state = useLaunch((s) => s.status?.state ?? '')
  const { data } = useLoaded<readonly Run[] | null>(
    game && profile ? () => Runs(game, profile) : null,
    [game, profile, crashId, state],
    null,
  )
  const block = game && profile ? bisectBlock(data) : 'no-profile'
  return block === null ? null : blockText(block, lang)
}

// findCrashCause asks before the crash check or, when there is none to run, says why in a toast.
export function findCrashCause(): void {
  const { game, openId } = useProfiles.getState()
  blockOf(game?.id, openId)
    .then((block) => {
      if (block === null) {
        useCommandPalette.setState({ confirmingBisect: true })
      } else {
        useToasts.getState().push({ kind: 'info', title: blockText(block, i18n) })
      }
    })
    .catch(reportUnexpected)
}
