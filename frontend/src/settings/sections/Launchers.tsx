import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import type { StoreApp } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { Launchers as ListLaunchers } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { useRefreshOnFocus } from '../../firstrun/useRefreshOnFocus.ts'
import { LauncherList } from '../../launchers/LauncherList.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { Searchable } from '../SettingsSection.tsx'

export function Launchers() {
  const { t } = useLingui()
  const [launchers, setLaunchers] = useState<StoreApp[]>([])
  const refresh = useCallback(() => {
    ListLaunchers()
      .then((ls) => setLaunchers(ls ?? []))
      .catch(reportUnexpected)
  }, [])
  useEffect(refresh, [refresh])
  useRefreshOnFocus(refresh)
  return (
    <Searchable
      terms={`${t`Launchers`} Steam GOG Flatpak ${t`game folder`} ${launchers.map((l) => l.name).join(' ')}`}
      loose={true}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
        <Box sx={{ fontSize: 13, color: 'var(--mortar-ink-sec)' }}>
          {t`Mortar finds your games through these launchers. Choose a folder for one it did not find, or that you moved.`}
        </Box>
        <LauncherList launchers={launchers} refresh={refresh} />
      </Box>
    </Searchable>
  )
}
