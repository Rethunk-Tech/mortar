import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, FormControl, InputLabel, MenuItem, Select, Typography } from '@mui/material'
import { useGameInfo } from '../games/info.ts'
import { useNexus } from '../settings/nexus.ts'
import { prefCopy } from '../settings/prefCopy.ts'
import {
  applyRow,
  choiceFromOverride,
  OVERRIDE_KEYS,
  OVERRIDE_VALUES,
  type OverrideKey,
  overrideChoiceLabel,
} from './overrideValue.ts'

const CONSOLE_KEYS = ['consoleLogCap', 'consoleLevel', 'consoleTimestamps', 'consoleFollow']
const REQUIREMENT_KEYS = ['enableRequirements', 'missingRequirements']

/** Whether a setting that depends on what the loader offers applies to this game's loader. */
function applies(
  key: OverrideKey,
  loader: { builtin: boolean; assets: boolean; console: boolean } | undefined,
): boolean {
  if (CONSOLE_KEYS.includes(key)) {
    return loader?.console === true
  }
  if (REQUIREMENT_KEYS.includes(key)) {
    return loader?.builtin !== true
  }
  return key !== 'conflictScanDepth' || loader?.assets === true
}

function overrideLabel(key: OverrideKey, i18n: I18n): string {
  switch (key) {
    case 'defaultLaunchMethod':
      return i18n._(msg`Launch method`)
    case 'showSmapiConsole':
      return i18n._(msg`SMAPI console window`)
    case 'backupBeforePlay':
      return i18n._(msg`Back up saves before Play`)
    case 'saveBackupsKept':
      return i18n._(msg`Save backups kept`)
    case 'updateModsBeforePlayDefault':
      return i18n._(msg`Update mods before Play`)
    case 'skipPlayCheck':
      return i18n._(msg`Skip pre-Play check`)
    case 'skipIntro':
      return i18n._(msg`Skip the intro`)
    case 'graphicsApi':
      return i18n._(msg`Graphics API`)
    case 'gameSettingsMode':
      return i18n._(msg`Game settings file`)
    default:
      return prefCopy(i18n, key).label ?? key
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
  const premium = useNexus((s) => s.premium)
  const info = useGameInfo()
  const smapi = info?.loaderId === 'smapi'
  const introSkip = info?.loaders?.[0]?.introSkip === true
  const graphics = info?.graphics ?? undefined
  const nexus = (info?.sources ?? []).includes('nexus')
  return (
    <Box sx={{ mt: 2, mb: 2 }}>
      <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{t`Overrides`}</Typography>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', mb: 1 }}>
        {t`These apply only to this profile.`}
      </Typography>
      {OVERRIDE_KEYS.filter(
        (key) =>
          (smapi || key !== 'showSmapiConsole') &&
          (introSkip || key !== 'skipIntro') &&
          (graphics || key !== 'graphicsApi') &&
          (info?.hasOptions || key !== 'gameSettingsMode') &&
          (info?.hasCaches || key !== 'cacheClearing') &&
          applies(key, info?.loaders?.[0]),
      ).map((key) => {
        const stored = overrides[key]
        const choice = choiceFromOverride(stored)
        const label = overrideLabel(key, i18n)
        const field = (
          <FormControl key={key} fullWidth={true} margin="dense" size="small">
            <InputLabel shrink={true} id={`override-${key}`}>
              {label}
            </InputLabel>
            <Select
              labelId={`override-${key}`}
              displayEmpty={true}
              notched={true}
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
              <MenuItem value="">{t`Use default`}</MenuItem>
              {(key === 'graphicsApi'
                ? (graphics?.choices ?? []).map((c) => ({ value: c.id, label: c.label }))
                : OVERRIDE_VALUES[key].map((value) => ({
                    value,
                    label: overrideChoiceLabel(key, value, i18n),
                  }))
              ).map((o) => (
                <MenuItem key={o.value} value={o.value}>
                  {o.label}
                </MenuItem>
              ))}
            </Select>
            {key === 'updateModsBeforePlayDefault' && nexus && !premium ? (
              <Typography sx={{ fontSize: 12, color: 'text.secondary', mt: 0.5 }}>
                {t`Free Nexus accounts click each download on Nexus; those updates do not block Play.`}
              </Typography>
            ) : null}
          </FormControl>
        )
        return field
      })}
    </Box>
  )
}
