import { msg } from '@lingui/core/macro'
import { StartVanilla } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { toastError } from '../toasts/report.ts'

const reportFailure = (error: unknown) => toastError(i18n._(msg`Could not launch the game`), error)

export async function startVanillaGame(opts: {
  get: () => { starting: boolean }
  set: (p: { starting: boolean; startingProfile?: string }) => void
  game: string
  direct: boolean
}) {
  if (opts.get().starting) {
    return
  }
  opts.set({ starting: true, startingProfile: '' })
  try {
    await StartVanilla(opts.game, opts.direct)
  } catch (error) {
    opts.set({ starting: false, startingProfile: '' })
    reportFailure(error)
  }
}
