import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, FormControlLabel } from '@mui/material'
import { PowerOff } from 'lucide-react'
import { useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { listNames } from '../i18n/list.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { disableFallout } from './disableUnneeded.ts'
import { LockedReason } from './LockedReason.tsx'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

// Asks before switching mods off when mods still on need them or libraries only they needed would stay on.
export function DisableDialog() {
  const disabling = useMods((s) => s.disabling)
  // The body subscribes to the mod list and profile; a closed dialog renders none of it.
  return disabling.length > 0 ? <DisableDialogBody disabling={disabling} /> : null
}

function DisableDialogBody({ disabling }: { disabling: Mod[] }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const askDisable = useMods((s) => s.askDisable)
  const disableMany = useMods((s) => s.disableMany)
  const locked = useLocked()
  const [withDependents, setWithDependents] = useState(true)
  const [withUnneeded, setWithUnneeded] = useState(true)
  const { dependents, unneeded } = disableFallout(mods, disabling, withDependents)
  const one = disabling.length === 1 ? disabling[0] : null
  const list = [
    ...disabling,
    ...(withDependents ? dependents : []),
    ...(withUnneeded ? unneeded : []),
  ]
  const close = () => askDisable(null)
  return (
    <AppDialog
      open={true}
      onClose={close}
      title={
        one
          ? t`Disable ${{ name: one.name }}?`
          : plural({ count: disabling.length }, { one: 'Disable # mod?', other: 'Disable # mods?' })
      }
      subtitle={
        dependents.length > 0
          ? plural(
              { count: disabling.length },
              {
                one: 'Mods that need it stop working while it is off.',
                other: 'Mods that need them stop working while they are off.',
              },
            )
          : plural(
              { count: disabling.length },
              {
                one: 'Nothing else needs the libraries it brought in.',
                other: 'Nothing else needs the libraries they brought in.',
              },
            )
      }
      icon={
        <ToneIcon tone="warning">
          <PowerOff size={22} />
        </ToneIcon>
      }
      footer={
        <DialogFooter>
          <Button onClick={close}>{t`Cancel`}</Button>
          <LockedReason locked={locked}>
            <Button
              variant="contained"
              disabled={locked}
              onClick={() => {
                disableMany(list).catch(reportUnexpected)
              }}
            >
              {list.length > disabling.length
                ? plural({ count: list.length }, { one: 'Disable # mod', other: 'Disable # mods' })
                : t`Disable`}
            </Button>
          </LockedReason>
        </DialogFooter>
      }
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
        {dependents.length > 0 ? (
          <FormControlLabel
            control={
              <Checkbox
                checked={withDependents}
                onChange={(e) => setWithDependents(e.target.checked)}
              />
            }
            label={t`${plural({ count: dependents.length }, { one: 'Also disable # mod that needs it', other: 'Also disable # mods that need it' })}: ${{ names: listNames(dependents.map((m) => m.name)) }}`}
          />
        ) : null}
        {unneeded.length > 0 ? (
          <FormControlLabel
            control={
              <Checkbox
                checked={withUnneeded}
                onChange={(e) => setWithUnneeded(e.target.checked)}
              />
            }
            label={t`${plural({ count: unneeded.length }, { one: 'Also disable # library nothing else needs', other: 'Also disable # libraries nothing else needs' })}: ${{ names: listNames(unneeded.map((m) => m.name)) }}`}
          />
        ) : null}
      </Box>
    </AppDialog>
  )
}
