import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import type { StoreApp } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import { Launchers } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import { ConfirmLaunchers } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { LauncherList } from '../launchers/LauncherList.tsx'
import { useNav } from '../nav/store.ts'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useRefreshOnFocus } from './useRefreshOnFocus.ts'

// First run: the launchers that tell Mortar which games are installed. Each game is set up when first opened.
export function FirstRun() {
  const { t } = useLingui()
  const [launchers, setLaunchers] = useState<StoreApp[] | null>(null)
  const [loadError, setLoadError] = useState('')
  const [busy, setBusy] = useState(false)
  const refresh = useCallback(() => {
    setLoadError('')
    setLaunchers(null)
    Launchers()
      .then((ls) => setLaunchers(ls ?? []))
      .catch((e: unknown) => setLoadError(errorMessage(e)))
  }, [])
  useEffect(refresh, [refresh])
  useRefreshOnFocus(refresh)
  if (loadError !== '') {
    return (
      <Box sx={{ p: 3, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1 }}>
        <Typography role="alert">{loadError}</Typography>
        <Button variant="contained" onClick={refresh}>
          {t`Retry`}
        </Button>
      </Box>
    )
  }
  if (!launchers) {
    return <LoadingRow>{t`Loading…`}</LoadingRow>
  }
  const found = launchers.filter((l) => l.found).length
  const finish = () => {
    setBusy(true)
    ConfirmLaunchers()
      .then(() => useNav.getState().openGameSelect())
      .catch((e: unknown) => {
        setBusy(false)
        reportUnexpected(e)
      })
  }
  return (
    <Box
      sx={{
        height: '100%',
        overflowY: 'auto',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: '20px',
        pt: '34px',
        pb: 3,
        px: 2,
        // Centred when it fits, scrolling from the top when it does not.
        '& > :first-of-type': { mt: 'auto' },
        '& > :last-child': { mb: 'auto' },
      }}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '6px' }}>
        <Typography component="h1" sx={{ fontSize: 40, fontWeight: 700 }}>
          {t`Welcome to Mortar`}
        </Typography>
        <Typography sx={{ fontSize: 17, textAlign: 'center' }}>
          {t`Mortar finds your games through the launchers that installed them. You set up each game when you first open it.`}
        </Typography>
      </Box>
      <Box sx={{ width: 'min(1200px, 100%)' }}>
        <LauncherList launchers={launchers} refresh={refresh} />
      </Box>
      <Box
        sx={{
          width: 'min(1200px, 100%)',
          display: 'flex',
          alignItems: 'center',
          gap: 2,
          justifyContent: 'flex-end',
        }}
      >
        <Typography sx={{ fontSize: 14, color: 'text.secondary', flex: 1 }}>
          {found > 0
            ? t`${found} of ${plural(launchers.length, { one: '# launcher', other: '# launchers' })} found.`
            : t`No launchers found. You can still continue and choose each game's folder when you open it.`}
        </Typography>
        <Button variant="contained" disabled={busy} onClick={finish} size="large" sx={{ px: 4 }}>
          {t`Continue`}
        </Button>
      </Box>
    </Box>
  )
}
