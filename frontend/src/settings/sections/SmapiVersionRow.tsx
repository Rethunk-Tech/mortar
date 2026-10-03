import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { useEffect, useState } from 'react'
import { State } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import {
  Install,
  InstallVersion,
  ListVersions,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/loadersvc/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useLaunch } from '../../launch/store.ts'
import { useLoader } from '../../loader/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { errorDetails, errorMessage } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSelect } from '../PrefControls.tsx'
import { persist } from '../persist.ts'
import { GAME_STARDEW } from '../prefValue.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

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

function SmapiVersionRow() {
  const { t, i18n } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  const pin = useSettings((s) => s.games?.stardew?.smapiPin)
  const status = useLoader((s) => s.status)
  const check = useLoader((s) => s.check)
  const pending = useLoader((s) => s.pending)
  const installing = useLoader((s) => s.installing)
  const playing = useLaunch(
    (s) =>
      s.starting ||
      (s.status?.game === GAME_STARDEW &&
        (s.status.state === State.Launching || s.status.state === State.Running)),
  )
  const [versions, setVersions] = useState<string[]>([])
  const [confirm, setConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  const selected = pinValue(pin)
  const locked = playing || pending || installing
  const lockReason = t`Stop the game to change mods.`
  useEffect(() => {
    ListVersions(GAME_STARDEW)
      .then((rows) => setVersions(rows ?? []))
      .catch(() => setVersions([]))
  }, [])
  const current = status?.version ? t`SMAPI ${status.version}` : t`SMAPI is not installed`
  const savePin = (value: string) => {
    const stored = value === LATEST ? '' : value
    persist(() => SetByKey('smapiPin', stored, GAME_STARDEW), push, fail)
  }
  const install = () => {
    setBusy(true)
    const run =
      selected === LATEST
        ? () => Install(GAME_STARDEW)
        : () => InstallVersion(GAME_STARDEW, selected)
    run()
      .then((st) => {
        push({ kind: 'success', title: t`SMAPI ${st.version} is installed` })
        return check(GAME_STARDEW)
      })
      .catch((err: unknown) => {
        const body = errorMessage(err)
        push({
          kind: 'error',
          title: t`Could not install SMAPI`,
          body,
          detail: errorDetails(err),
        })
      })
      .finally(() => {
        setBusy(false)
        setConfirm(false)
      })
  }
  return (
    <>
      <SettingRow label={t`SMAPI version`} description={current}>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
          <PrefSelect
            value={selected}
            onChange={savePin}
            options={versionChoices(i18n, pin ? [pin, ...versions] : versions)}
          />
          <DisabledReason title={lockReason} disabled={locked}>
            <Button
              variant="outlined"
              disabled={locked}
              onClick={() => setConfirm(true)}
              sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
            >
              {t`Install this version`}
            </Button>
          </DisabledReason>
        </Box>
      </SettingRow>
      <ConfirmDialog
        open={confirm}
        title={t`Install this SMAPI version?`}
        body={
          selected === LATEST
            ? t`Install the latest SMAPI and follow new releases.`
            : t`Install SMAPI ${selected}.`
        }
        confirmLabel={t`Install`}
        busy={busy}
        onCancel={() => setConfirm(false)}
        onConfirm={install}
      />
    </>
  )
}

export { SmapiVersionRow }
