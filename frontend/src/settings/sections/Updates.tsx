import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, CircularProgress, Switch } from '@mui/material'
import { Download, RefreshCw, RotateCcw } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import {
  SetCheckOnlyEnabledMods,
  SetIncludeBetaReleases,
  SetIncludePrereleaseModVersions,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useNav } from '../../nav/store.ts'
import { errorMessage, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { useMortarUpdate } from '../updates.ts'
import { lastOpenedGame } from './lastOpenedGame.ts'

const button = { whiteSpace: 'nowrap', flexShrink: 0 }

function MortarUpdate() {
  const { t } = useLingui()
  const { info, phase, release, error, load, check, install, restart } = useMortarUpdate()
  const [failed, setFailed] = useState('')
  const read = useCallback(() => {
    setFailed('')
    load().catch((e: unknown) => setFailed(errorMessage(e)))
  }, [load])
  useEffect(read, [read])
  if (!info) {
    return failed ? (
      <Alert
        severity="error"
        action={
          <Button color="inherit" size="small" onClick={read} sx={{ whiteSpace: 'nowrap' }}>
            {t`Retry`}
          </Button>
        }
        sx={{ fontSize: 14 }}
      >
        {t`Could not read this Mortar build's version: ${failed}`}
      </Alert>
    ) : null
  }
  if (info.off === 'packaged') {
    return (
      <Alert severity="info" sx={{ fontSize: 14 }}>
        {t`Updated by your package manager`}
      </Alert>
    )
  }
  if (info.off) {
    return (
      <Alert severity="info" sx={{ fontSize: 14 }}>
        {t`This is a development build of Mortar ${info.version}, so it does not check for updates.`}
      </Alert>
    )
  }
  const busy = phase === 'checking' || phase === 'installing' || phase === 'restarting'
  const states: Record<typeof phase, string> = {
    idle: t`Not checked yet.`,
    checking: t`Checking for updates…`,
    current: t`Mortar is up to date.`,
    none: t`No release is published yet.`,
    available: t`Mortar ${release?.version ?? ''} is available.`,
    installing: t`Downloading and verifying Mortar ${release?.version ?? ''}…`,
    ready: t`Update ready — applies when you close Mortar.`,
    restarting: t`Restarting…`,
    error,
  }
  let action = (
    <Button
      variant="outlined"
      startIcon={<RefreshCw size={16} />}
      disabled={busy}
      onClick={() => check().catch(reportUnexpected)}
      sx={button}
    >
      {t`Check now`}
    </Button>
  )
  if (phase === 'available') {
    action = (
      <Button
        variant="contained"
        startIcon={<Download size={16} />}
        onClick={() => install().catch(reportUnexpected)}
        sx={button}
      >
        {t`Download and install`}
      </Button>
    )
  } else if (phase === 'ready' || phase === 'restarting') {
    action = (
      <Button
        variant="contained"
        startIcon={<RotateCcw size={16} />}
        disabled={busy}
        onClick={() => restart().catch(reportUnexpected)}
        sx={button}
      >
        {t`Restart now`}
      </Button>
    )
  }
  return (
    <>
      <SettingRow
        label={t`Installed: Mortar ${info.version}`}
        description={
          <Box
            component="span"
            role="status"
            sx={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 1,
              color: phase === 'error' ? 'error.main' : 'inherit',
            }}
          >
            {busy ? <CircularProgress size={12} /> : null}
            {states[phase]}
          </Box>
        }
      >
        {action}
      </SettingRow>
      {phase === 'available' && release?.notes ? (
        <Box
          sx={{
            fontSize: 13,
            whiteSpace: 'pre-wrap',
            maxHeight: 200,
            overflow: 'auto',
            px: 2,
            py: 1.5,
          }}
        >
          {release.notes}
        </Box>
      ) : null}
    </>
  )
}

export function Updates() {
  const { t } = useLingui()
  const game = lastOpenedGame(useSettings((s) => s.lastGame))
  const includeBetaReleases = useSettings((s) => s.includeBetaReleases)
  const includePrereleaseModVersions = useSettings((s) => s.includePrereleaseModVersions)
  const checkOnlyEnabledMods = useSettings((s) => s.checkOnlyEnabledMods)
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  return (
    <>
      <SettingsSection title={t`Mortar`}>
        <MortarUpdate />
        <SettingRow label={t`Include beta releases`}>
          <Switch
            checked={includeBetaReleases}
            onChange={(_, on) => persist(() => SetIncludeBetaReleases(on), push, fail)}
          />
        </SettingRow>
        <PrefByKey prefKey="autoInstallMortarUpdates" />
      </SettingsSection>
      <SettingsSection title={t`Mods`}>
        <PrefKeys keys={['checkModUpdatesOnStart', 'updateCheckIntervalMinutes']} />
        <SettingRow label={t`Check only enabled mods`}>
          <Switch
            checked={checkOnlyEnabledMods}
            onChange={(_, on) => persist(() => SetCheckOnlyEnabledMods(on), push, fail)}
          />
        </SettingRow>
        <SettingRow label={t`Include pre-release mod versions`}>
          <Switch
            checked={includePrereleaseModVersions}
            onChange={(_, on) => persist(() => SetIncludePrereleaseModVersions(on), push, fail)}
          />
        </SettingRow>
        {game ? (
          <SettingRow
            label={t`Review mod updates`}
            description={t`Mod updates show on each profile's mod list.`}
          >
            <Button variant="outlined" onClick={() => useNav.getState().openGame(game)} sx={button}>
              {t`Review`}
            </Button>
          </SettingRow>
        ) : null}
        {game ? (
          <SettingRow
            label={t`Mod loader updates`}
            description={t`A game's mod loader version and its update notice are in that game's settings.`}
          >
            <Button
              variant="outlined"
              onClick={() => useNav.setState({ route: { name: 'game-settings', game } })}
              sx={button}
            >
              {t`Open game settings`}
            </Button>
          </SettingRow>
        ) : null}
      </SettingsSection>
    </>
  )
}
