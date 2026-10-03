import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, CircularProgress, FormControlLabel, Link, Switch } from '@mui/material'
import { Download, RefreshCw, RotateCcw } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { List as ListGames } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import {
  SetAutoInstallMortarUpdates,
  SetCheckModUpdatesOnStart,
  SetCheckOnlyEnabledMods,
  SetIncludeBetaReleases,
  SetIncludePrereleaseModVersions,
  SetNotifyModUpdates,
  SetSmapiBuilds,
  SetUpdateCheckIntervalMinutes,
  SetUpdateModsBeforePlayDefault,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useNav } from '../../nav/store.ts'
import { errorMessage, errorText, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { gamePrefs } from '../gamePrefs.ts'
import { PrefNumber, PrefSelect, PrefSwitch } from '../PrefControls.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { useMortarUpdate } from '../updates.ts'
import { lastOpenedGame } from './lastOpenedGame.ts'

const button = { whiteSpace: 'nowrap', alignSelf: 'flex-start' }

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
      <Box
        sx={{ fontSize: 14, color: 'rgba(225,225,230,0.95)' }}
      >{t`Installed: Mortar ${info.version}`}</Box>
      <Box
        role="status"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          fontSize: 14,
          color: phase === 'error' ? 'error.main' : 'inherit',
        }}
      >
        {busy ? <CircularProgress size={14} /> : null}
        {states[phase]}
      </Box>
      {phase === 'available' && release?.notes ? (
        <Box
          sx={{
            fontSize: 13,
            whiteSpace: 'pre-wrap',
            maxHeight: 200,
            overflow: 'auto',
            p: 1.5,
            bgcolor: 'rgba(0,0,0,0.3)',
            borderRadius: '6px',
          }}
        >
          {release.notes}
        </Box>
      ) : null}
      {action}
    </>
  )
}

function persistToggle(
  run: () => Promise<void>,
  push: ReturnType<typeof useToasts.getState>['push'],
  title: string,
) {
  run().catch((err: unknown) => {
    const body = errorText(err)
    push({ kind: 'error', title, ...(body ? { body } : {}) })
  })
}

export function Updates() {
  const { t } = useLingui()
  const game = lastOpenedGame(useSettings((s) => s.lastGame))
  const [gameName, setGameName] = useState('')
  useEffect(() => {
    if (!game) {
      return
    }
    ListGames()
      .then((gs) => setGameName((gs ?? []).find((g) => g.id === game)?.name ?? ''))
      .catch(() => setGameName(''))
  }, [game])
  const checkModUpdatesOnStart = useSettings((s) => s.checkModUpdatesOnStart)
  const includeBetaReleases = useSettings((s) => s.includeBetaReleases)
  const includePrereleaseModVersions = useSettings((s) => s.includePrereleaseModVersions)
  const checkOnlyEnabledMods = useSettings((s) => s.checkOnlyEnabledMods)
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Mortar`}</Box>
      <MortarUpdate />
      <FormControlLabel
        sx={{ m: 0, alignItems: 'center' }}
        control={
          <Switch
            checked={includeBetaReleases}
            onChange={(_, on) => persistToggle(() => SetIncludeBetaReleases(on), push, fail)}
          />
        }
        label={t`Include beta releases`}
      />
      <SettingRow
        label={t`Install Mortar updates automatically`}
        description={t`Download and stage a found update without asking`}
      >
        <PrefSwitch
          checked={useSettings((s) => s.autoInstallMortarUpdates) !== false}
          onChange={(on) => persist(() => SetAutoInstallMortarUpdates(on), push, fail)}
        />
      </SettingRow>
      <Box sx={{ fontSize: 14, fontWeight: 600, pt: 1 }}>{t`Mods`}</Box>
      <SettingsSection>
        <SettingRow
          label={t`Check interval`}
          description={t`Minutes between background update checks`}
        >
          <PrefNumber
            value={useSettings((s) => s.updateCheckIntervalMinutes) || 60}
            min={15}
            max={1440}
            onCommit={(n) => SetUpdateCheckIntervalMinutes(n)}
          />
        </SettingRow>
        <SettingRow label={t`Notify when updates are found`}>
          <PrefSwitch
            checked={Boolean(useSettings((s) => s.notifyModUpdates))}
            onChange={(on) => persist(() => SetNotifyModUpdates(on), push, fail)}
          />
        </SettingRow>
        <SettingRow
          label={t`Update mods before Play on new profiles`}
          description={t`Default for a profile you just created`}
        >
          <PrefSwitch
            checked={useSettings((s) => gamePrefs(s).updateModsBeforePlayDefault)}
            onChange={(on) => persist(() => SetUpdateModsBeforePlayDefault(on), push, fail)}
          />
        </SettingRow>
      </SettingsSection>
      <FormControlLabel
        sx={{ m: 0, alignItems: 'center' }}
        control={
          <Switch
            checked={checkModUpdatesOnStart !== false}
            onChange={(_, on) => persistToggle(() => SetCheckModUpdatesOnStart(on), push, fail)}
          />
        }
        label={t`Check for mod updates when Mortar starts`}
      />
      <FormControlLabel
        sx={{ m: 0, alignItems: 'center' }}
        control={
          <Switch
            checked={includePrereleaseModVersions}
            onChange={(_, on) =>
              persistToggle(() => SetIncludePrereleaseModVersions(on), push, fail)
            }
          />
        }
        label={t`Include pre-release mod versions`}
      />
      <SettingsSection>
        <SettingRow
          label={t`SMAPI unofficial builds`}
          description={t`Pre-releases and unofficial SMAPI updates. Show lists them; Include lets Update all install them.`}
        >
          <PrefSelect
            value={useSettings((s) => gamePrefs(s).smapiBuilds) || 'show'}
            onChange={(v) => persist(() => SetSmapiBuilds(v), push, fail)}
            options={[
              { value: 'never', label: t`Never` },
              { value: 'show', label: t`Show` },
              { value: 'include', label: t`Include` },
            ]}
          />
        </SettingRow>
      </SettingsSection>
      <FormControlLabel
        sx={{ m: 0, alignItems: 'center' }}
        control={
          <Switch
            checked={checkOnlyEnabledMods}
            onChange={(_, on) => persistToggle(() => SetCheckOnlyEnabledMods(on), push, fail)}
          />
        }
        label={t`Check only enabled mods`}
      />
      <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
        {t`Mod updates show on each profile's mod list. A game's mod loader version and its update notice are in that game's settings.`}
      </Box>
      {game ? (
        <>
          <Link
            component="button"
            onClick={() => useNav.getState().openGame(game)}
            sx={{ alignSelf: 'flex-start', fontSize: 13, whiteSpace: 'nowrap' }}
          >
            {t`Review mod updates`}
          </Link>
          <Link
            component="button"
            onClick={() => useNav.setState({ route: { name: 'game-settings', game } })}
            sx={{ alignSelf: 'flex-start', fontSize: 13, whiteSpace: 'nowrap' }}
          >
            {gameName ? t`Open ${gameName} settings` : t`Open the game's settings`}
          </Link>
        </>
      ) : null}
    </Box>
  )
}
