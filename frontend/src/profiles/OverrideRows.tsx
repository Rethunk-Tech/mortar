import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  FormControl,
  FormHelperText,
  InputLabel,
  MenuItem,
  Select,
  Typography,
} from '@mui/material'
import { useGameInfo } from '../games/info.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useNexus } from '../settings/nexus.ts'
import { prefCopy } from '../settings/prefCopy.ts'
import { useProtonBuilds } from '../settings/protonBuilds.ts'
import { useSettings } from '../settings/store.ts'
import { space } from '../theme/density.ts'
import {
  applyRow,
  choiceFromOverride,
  OVERRIDE_KEYS,
  OVERRIDE_VALUES,
  type OverrideKey,
  overrideChoiceLabel,
  resolveOverride,
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
    case 'protonVersion':
      return i18n._(msg`Proton version`)
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
  const launch = useSettings((s) => gamePrefs(s, info?.id).defaultLaunchMethod)
  const directPlay = resolveOverride('defaultLaunchMethod', launch, overrides) === 'direct'
  const smapi = info?.loaderId === 'smapi'
  const introSkip = info?.loaders?.[0]?.introSkip === true
  const graphics = info?.graphics ?? undefined
  const proton = useProtonBuilds(info?.id ?? '')
  const nexus = (info?.sources ?? []).includes('nexus')
  // The game's catalog and this computer's Proton builds supply two keys' choices; the rest are fixed.
  const choicesFor = (key: OverrideKey) => {
    if (key === 'graphicsApi') {
      return (graphics?.choices ?? []).map((c) => ({ value: c.id, label: c.label }))
    }
    if (key === 'protonVersion') {
      return proton.map((b) => ({ value: b, label: b }))
    }
    return OVERRIDE_VALUES[key].map((value) => ({
      value,
      label: overrideChoiceLabel(key, value, i18n),
    }))
  }
  return (
    <Box sx={{ mt: 2, mb: 2 }}>
      <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{t`Overrides`}</Typography>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', mb: 1 }}>
        {t`These apply only to this profile.`}
      </Typography>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.pad, pt: 1 }}>
        {OVERRIDE_KEYS.filter(
          (key) =>
            (smapi || key !== 'showSmapiConsole') &&
            (introSkip || key !== 'skipIntro') &&
            (graphics || key !== 'graphicsApi') &&
            (proton.length > 0 || key !== 'protonVersion') &&
            (info?.hasOptions || key !== 'gameSettingsMode') &&
            (info?.hasCaches || key !== 'cacheClearing') &&
            (info?.sources?.includes('thunderstore') || key !== 'dependencyVersions') &&
            (directPlay || (key !== 'instances' && key !== 'delaySeconds')) &&
            applies(key, info?.loaders?.[0]),
        ).map((key) => {
          const stored = overrides[key]
          const choice = choiceFromOverride(stored)
          const label = overrideLabel(key, i18n)
          const field = (
            <FormControl key={key} fullWidth={true} size="small">
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
                {choicesFor(key).map((o) => (
                  <MenuItem key={o.value} value={o.value}>
                    {o.label}
                  </MenuItem>
                ))}
              </Select>
              {key === 'updateModsBeforePlayDefault' && nexus && !premium ? (
                <FormHelperText>
                  {t`Free Nexus accounts click each download on Nexus; those updates do not block Play.`}
                </FormHelperText>
              ) : null}
            </FormControl>
          )
          return field
        })}
      </Box>
    </Box>
  )
}
