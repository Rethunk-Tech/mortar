import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import { ListVersions } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/loadersvc/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useGameBusy, useLaunch } from '../../launch/store.ts'
import { InstallSteps } from '../../loader/InstallSteps.tsx'
import { useLoader } from '../../loader/store.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { useToasts } from '../../toasts/store.ts'
import { PrefSelect } from '../PrefControls.tsx'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { type SmapiAction, smapiAction } from './smapiAction.ts'

const LATEST = 'latest'

function versionChoices(i18n: I18n, versions: string[]): { value: string; label: string }[] {
  const seen = new Set<string>([LATEST])
  const options = [{ value: LATEST, label: i18n._(msg`Latest (recommended)`) }]
  for (const version of versions) {
    if (!seen.has(version)) {
      seen.add(version)
      options.push({ value: version, label: version })
    }
  }
  return options
}

function pinValue(raw: string | undefined): string {
  if (!raw) {
    return LATEST
  }
  return raw
}

function SmapiRow({ onVersion }: { onVersion: (v: string) => void }) {
  const { t, i18n } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Could not save that setting`
  const game = useCurrentGame()
  const pin = useSettings((s) => s.smapiPin)
  const status = useLoader((s) => s.status)
  const check = useLoader((s) => s.check)
  const install = useLoader((s) => s.install)
  const pending = useLoader((s) => s.pending)
  const installing = useLoader((s) => s.installing)
  const steps = useLoader((s) => s.steps)
  const refreshLaunch = useLaunch((s) => s.refresh)
  const playing = useGameBusy(game)
  const [versions, setVersions] = useState<string[]>([])
  const [confirm, setConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  const selected = pinValue(pin)
  useEffect(() => {
    check(game)
    refreshLaunch(game)
    ListVersions(game)
      .then((rows) => setVersions(rows ?? []))
      .catch(() => setVersions([]))
  }, [check, refreshLaunch, game])
  const gameVersion = status?.gameVersion ?? ''
  useEffect(() => {
    onVersion(gameVersion)
  }, [gameVersion, onVersion])
  const locked = playing || pending || installing
  const lockReason = playing
    ? t`Stop the game to install SMAPI.`
    : t`An install is already running.`
  const action = smapiAction(
    {
      installed: status?.installed === true,
      broken: status?.broken === true,
      version: status?.version ?? '',
      updateAvailable: status?.updateAvailable === true,
    },
    selected,
    LATEST,
  )
  const labels: Record<SmapiAction['kind'], string> = {
    install: action.version ? t`Install ${action.version}…` : t`Install`,
    update: t`Update`,
    reinstall: t`Reinstall`,
  }
  let title = t`SMAPI is not installed`
  let detail = ''
  if (status?.broken) {
    title = t`A game update replaced SMAPI's launcher`
  } else if (status?.installed) {
    title = t`SMAPI ${status.version}`
    detail = status.updateAvailable
      ? t`SMAPI ${status.latest} is available`
      : t`Installed and up to date`
  }
  const run = () => {
    setBusy(true)
    install(game, action.version || undefined).finally(() => {
      setBusy(false)
      setConfirm(false)
    })
  }
  const savePin = (value: string) => {
    persist(() => SetByKey('smapiPin', value === LATEST ? '' : value, game), push, fail)
  }
  return (
    <>
      <SettingRow label={title} description={detail} block={installing}>
        {installing ? (
          <InstallSteps steps={steps} />
        ) : (
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            <PrefSelect
              value={selected}
              onChange={savePin}
              options={versionChoices(i18n, pin ? [pin, ...versions] : versions)}
              label={t`SMAPI version`}
            />
            <DisabledReason title={lockReason} disabled={locked}>
              <Button
                variant="outlined"
                startIcon={<Download size={16} />}
                disabled={locked || status === null}
                onClick={() => (action.confirm ? setConfirm(true) : run())}
                sx={{ flexShrink: 0 }}
              >
                {labels[action.kind]}
              </Button>
            </DisabledReason>
          </Box>
        )}
      </SettingRow>
      <ConfirmDialog
        open={confirm}
        title={t`Install SMAPI ${action.version}?`}
        body={t`Every profile of this game runs the SMAPI in the game folder, so they all switch to ${action.version}.`}
        confirmLabel={t`Install`}
        busy={busy}
        onCancel={() => setConfirm(false)}
        onConfirm={run}
      />
    </>
  )
}

export { SmapiRow }
