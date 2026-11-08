import { useLingui } from '@lingui/react/macro'
import { useCallback } from 'react'
import { Launchers as ListLaunchers } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { LauncherList } from '../../launchers/LauncherList.tsx'
import { RefreshFailed } from '../../shell/RefreshFailed.tsx'
import { SkeletonRows } from '../../shell/SkeletonRows.tsx'
import { useLastGood } from '../../shell/useLastGood.ts'
import { useOnWindowFocus } from '@rethunk/react-load-hooks'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { Searchable, SettingsSection } from '../SettingsSection.tsx'

const SKELETON_LAUNCHERS = 10
const SKELETON_LAUNCHER_PX = 60

export function Launchers() {
  const { t } = useLingui()
  // null until the very first answer, so the page shows where the launchers will be rather than nothing.
  const {
    data: launchers,
    error,
    at,
    reload,
  } = useLastGood('launchers', () => ListLaunchers().then((ls) => ls ?? []), [], {
    onError: reportUnexpected,
  })
  const refresh = useCallback(() => {
    reload().catch(reportUnexpected)
  }, [reload])
  useOnWindowFocus(refresh)
  return (
    <SettingsSection
      title={t`Launchers`}
      description={t`Mortar finds your games through these launchers. Choose a folder for a game it missed or you moved.`}
      tiles={false}
    >
      <Searchable
        terms={`${t`Launchers`} Steam GOG Flatpak ${t`Game folder`} ${(launchers ?? []).map((l) => l.name).join(' ')}`}
      >
        {launchers ? (
          <>
            <RefreshFailed error={error} at={at} onRetry={refresh} />
            <LauncherList launchers={launchers} refresh={refresh} />
          </>
        ) : (
          <SkeletonRows
            label={t`Looking for launchers…`}
            count={SKELETON_LAUNCHERS}
            height={SKELETON_LAUNCHER_PX}
            sx={{
              display: 'grid',
              gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr)',
              gap: space.gap,
            }}
          />
        )}
      </Searchable>
    </SettingsSection>
  )
}
