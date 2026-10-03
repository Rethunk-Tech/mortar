import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, FormControl, InputLabel, MenuItem, Select, Typography } from '@mui/material'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useNexus } from '../settings/nexus.ts'
import { useSettings } from '../settings/store.ts'
import {
  applyRow,
  choiceFromOverride,
  gamePrefString,
  OVERRIDE_KEYS,
  OVERRIDE_VALUES,
  type OverrideKey,
} from './overrideValue.ts'

function overrideLabel(key: OverrideKey, i18n: I18n): string {
  switch (key) {
    case 'defaultLaunchMethod':
      return i18n._(msg`Launch method`)
    case 'showSmapiConsole':
      return i18n._(msg`Show SMAPI console`)
    case 'backupBeforePlay':
      return i18n._(msg`Backup before Play`)
    case 'launchBackupsKept':
      return i18n._(msg`Launch backups kept`)
    case 'updateModsBeforePlayDefault':
      return i18n._(msg`Update mods before Play`)
    case 'skipPlayCheck':
      return i18n._(msg`Skip pre-Play check`)
    default:
      return key
  }
}

export function OverridesSection({
  overrides,
  onChange,
}: {
  overrides: Record<string, string>
  onChange: (next: Record<string, string>) => void
}) {
  const { t, i18n } = useLingui()
  const prefs = gamePrefs(useSettings())
  const premium = useNexus((s) => s.premium)
  return (
    <Box sx={{ mt: 2, mb: 2 }}>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>{t`Overrides`}</Typography>
      {OVERRIDE_KEYS.map((key) => {
        const gameValue = gamePrefString(key, prefs)
        const stored = overrides[key]
        const choice = choiceFromOverride(stored)
        const label = overrideLabel(key, i18n)
        return (
          <FormControl key={key} fullWidth={true} margin="dense" size="small">
            <InputLabel id={`override-${key}`}>{label}</InputLabel>
            <Select
              labelId={`override-${key}`}
              label={label}
              value={choice.useGame ? '' : choice.value}
              onChange={(event) => {
                const next = event.target.value
                onChange(
                  applyRow(
                    overrides,
                    key,
                    next === '' ? { useGame: true } : { useGame: false, value: next },
                  ),
                )
              }}
            >
              <MenuItem value="">{t`Use game setting (${gameValue})`}</MenuItem>
              {OVERRIDE_VALUES[key].map((value) => (
                <MenuItem key={value} value={value}>
                  {value}
                </MenuItem>
              ))}
            </Select>
            {key === 'updateModsBeforePlayDefault' && !premium ? (
              <Typography sx={{ fontSize: 12, color: 'text.secondary', mt: 0.5 }}>
                {t`Free Nexus accounts must click each download on Nexus; those updates will not block Play.`}
              </Typography>
            ) : null}
          </FormControl>
        )
      })}
    </Box>
  )
}
