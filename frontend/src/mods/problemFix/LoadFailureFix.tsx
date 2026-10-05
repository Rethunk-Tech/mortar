import { useLingui } from '@lingui/react/macro'
import { useBrowseView } from '../../browse/view.ts'
import { useTab } from '../../game/tab.ts'
import { useLoader } from '../../loader/store.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { openTarget } from '../storeView.ts'
import { useUpdates } from '../updates.ts'
import type { WarningButton } from './warningButton.tsx'

// The fix follows the cause the log names: a missing plugin is found, one built for other versions is updated, a
// broken loader is reinstalled, and a plugin that clashes or keeps failing is disabled.
export function LoadFailureFix({
  problem,
  button,
}: {
  problem: Extract<Problem, { kind: 'loadFailure' }>
  button: WarningButton
}) {
  const { t } = useLingui()
  const failure = problem.loadFailure
  const owner = useMods((s) => s.mods.find((m) => failure.key !== '' && m.key === failure.key))
  switch (failure.kind) {
    case 'missing-dependency': {
      const dependency = failure.dependency ?? ''
      return dependency === ''
        ? null
        : button(t`Find ${dependency}`, () => {
            useBrowseView.getState().setPendingQuery(dependency)
            useTab.getState().setTab('browse')
          })
    }
    case 'incompatible-version':
    case 'load-exception':
    case 'patch-exception':
      return button(t`Update`, () => {
        useTab.getState().setTab('mods')
        useMods.getState().showUpdates()
        useUpdates.getState().setReviewing(true)
      })
    case 'preloader-patch':
    case 'loader-version':
    case 'chainloader':
      // A patcher that failed is the patcher's fault; only the preloader itself failing is the loader's.
      if (
        failure.kind === 'preloader-patch' &&
        !failure.message.startsWith('Could not run preloader')
      ) {
        return null
      }
      return button(t`Reinstall loader`, () => {
        const at = openTarget()
        if (at) {
          useLoader.getState().install(at.game).catch(reportUnexpected)
        }
      })
    default:
      return owner
        ? button(t`Disable`, () =>
            useMods.getState().setEnabled(owner, false).catch(reportUnexpected),
          )
        : null
  }
}
