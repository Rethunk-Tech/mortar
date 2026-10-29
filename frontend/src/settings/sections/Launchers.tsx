import { useLingui } from '@lingui/react/macro'
import { useCallback, useEffect, useState } from 'react'
import type { StoreApp } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { Launchers as ListLaunchers } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { useRefreshOnFocus } from '../../firstrun/useRefreshOnFocus.ts'
import { LauncherList } from '../../launchers/LauncherList.tsx'
import { SkeletonRows } from '../../shell/SkeletonRows.tsx'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { Searchable, SettingsSection } from '../SettingsSection.tsx'

const SKELETON_LAUNCHERS = 10
const SKELETON_LAUNCHER_PX = 60

export function Launchers() {
  const { t } = useLingui()
  // null until the first answer, so the page shows where the launchers will be rather than nothing.
  const [launchers, setLaunchers] = useState<StoreApp[] | null>(null)
  const refresh = useCallback(() => {
    ListLaunchers()
      .then((ls) => setLaunchers(ls ?? []))
      .catch(reportUnexpected)
  }, [])
  useEffect(refresh, [refresh])
  useRefreshOnFocus(refresh)
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
          <LauncherList launchers={launchers} refresh={refresh} />
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
