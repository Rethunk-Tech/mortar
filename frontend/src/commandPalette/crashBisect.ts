import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Start as StartBisect } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bisect/service.ts'
import type { Run } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { type BisectBlock, bisectBlock } from '../console/bisectBlock.ts'
import { i18n } from '../i18n/index.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useCommandPalette } from './store.ts'

// Built on call, not at import: Lingui macros only run inside compiled code.
function blockText(block: BisectBlock, lang: I18n): string {
  switch (block) {
    case 'no-profile':
      return lang._(msg`Open a profile first.`)
    case 'no-runs':
      return lang._(msg`Start the game once so Mortar can see the crash.`)
    case 'ended-normally':
      return lang._(msg`No crash to investigate: the last run ended normally.`)
    default:
      return lang._(msg`Mortar already named the likely mod for the last crash.`)
  }
}

async function blockOf(game: string | undefined, profile: string | undefined) {
  if (!(game && profile)) {
    return 'no-profile'
  }
  return bisectBlock(await Runs(game, profile))
}

async function startCrashBisect(): Promise<string | null> {
  const { game, openId } = useProfiles.getState()
  const block = await blockOf(game?.id, openId)
  if (block !== null) {
    return blockText(block, i18n)
  }
  const id = await StartBisect(game?.id ?? '', openId ?? '')
  useCommandPalette.getState().setBisect({ id, game: game?.id ?? '', profile: openId ?? '' })
  return null
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

// findCrashCause starts the bisect or, when it cannot, says why in a toast.
export function findCrashCause(): void {
  startCrashBisect()
    .then((message) => {
      if (message) {
        useToasts.getState().push({ kind: 'info', title: message })
      }
    })
    .catch(reportUnexpected)
}
