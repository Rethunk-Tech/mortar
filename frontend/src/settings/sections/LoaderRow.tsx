import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import { ListVersions } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/loadersvc/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useGameBusy, useLaunch } from '../../launch/store.ts'
import { InstallSteps } from '../../loader/InstallSteps.tsx'
import { useLoader } from '../../loader/store.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { ConfirmationDialog } from '../../shell/dialog/ConfirmationDialog.tsx'
import { space } from '../../theme/density.ts'
import { PrefSelect } from '../PrefControls.tsx'
import { saveSetting } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { useSettingsMatch } from '../useSettingsSearch.ts'
import { type SmapiAction, smapiAction } from './smapiAction.ts'

const LATEST = 'latest'

const NO_VERSIONS: string[] = []

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

function LoaderRow({
  loader,
  onVersion,
}: {
  loader: { id: string; name: string }
  onVersion: (v: string) => void
}) {
  const { t, i18n } = useLingui()
  const game = useCurrentGame()
  const pin = useSettings((s) => s.loaderPrefs?.[`${game}/${loader.id}`]?.pin ?? '')
  const status = useLoader((s) => s.status)
  const check = useLoader((s) => s.check)
  const install = useLoader((s) => s.install)
  const pending = useLoader((s) => s.pending)
  const installing = useLoader((s) => s.installing)
  const steps = useLoader((s) => s.steps)
  const refreshLaunch = useLaunch((s) => s.refresh)
  const playing = useGameBusy(game)
  const [confirm, setConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  const selected = pinValue(pin)
  useEffect(() => {
    check(game)
  }, [check, game])
  const gameVersion = status?.gameVersion ?? ''
  useEffect(() => {
    onVersion(gameVersion)
  }, [gameVersion, onVersion])
  const locked = playing || pending || installing
  const lockReason = playing
    ? t`Stop the game to install ${{ name: loader.name }}.`
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
    install: action.version ? t`Install ${{ version: action.version }}…` : t`Install`,
    update: t`Update`,
    reinstall: t`Reinstall`,
  }
  let title = t`${{ name: loader.name }} is not installed`
  let detail = ''
  if (status?.broken) {
    title = t`A game update replaced ${{ name: loader.name }}'s launcher`
  } else if (status?.installed) {
    title = t`${{ name: loader.name }} ${{ version: status.version }}`
    detail = status.updateAvailable
      ? t`${{ name: loader.name }} ${{ latest: status.latest }} is available`
      : t`Installed and up to date`
  }
  // The status above is what a search matches; the launch state and the version list only feed the controls.
  const relevant = useSettingsMatch(title, detail)
  useEffect(() => {
    if (!relevant) {
      return
    }
    refreshLaunch(game)
  }, [relevant, refreshLaunch, game])
  const { data: versions } = useLoaded(
    relevant
      ? () =>
          ListVersions(game, loader.id).then(
            (rows) => rows ?? NO_VERSIONS,
            () => NO_VERSIONS,
          )
      : null,
    [relevant, game, loader.id],
    NO_VERSIONS,
  )
  const run = () => {
    setBusy(true)
    install(game, action.version || undefined).finally(() => {
      setBusy(false)
      setConfirm(false)
    })
  }
  const savePin = (value: string) => {
    saveSetting(() => SetByKey(`${loader.id}Pin`, value === LATEST ? '' : value, game))
  }
  return (
    <>
      <SettingRow label={title} description={detail} block={installing}>
        {installing ? (
          <InstallSteps steps={steps} />
        ) : (
          <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'center' }}>
            <PrefSelect
              value={selected}
              onChange={savePin}
              options={versionChoices(i18n, pin ? [pin, ...versions] : versions)}
              label={t`${{ name: loader.name }} version`}
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
      <ConfirmationDialog
        open={confirm}
        title={t`Install ${{ name: loader.name }} ${{ version: action.version }}?`}
        body={t`Every profile of this game runs the ${{ name: loader.name }} installed for it, so they all switch to ${{ version: action.version }}.`}
        confirmLabel={t`Install`}
        busy={busy}
        onCancel={() => setConfirm(false)}
        onConfirm={run}
      />
    </>
  )
}

export { LoaderRow }
