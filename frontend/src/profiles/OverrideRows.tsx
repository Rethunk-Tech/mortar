import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, MenuItem, Typography } from '@mui/material'
import { PROTON, SMAPI } from '../brand/productNames.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useGameInfo } from '../games/info.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useNexus } from '../settings/nexus.ts'
import { prefCopy } from '../settings/prefCopy.ts'
import { useProtonBuilds } from '../settings/protonBuilds.ts'
import { useSettings } from '../settings/store.ts'
import { SelectField } from '../shell/dialog/FormDialog.tsx'
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
    case 'startFirst':
      return i18n._(msg`Start first`)
    case 'showSmapiConsole':
      return i18n._(msg`${{ loader: SMAPI }} console window`)
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
      return i18n._(msg`${{ name: PROTON }} version`)
    case 'gameSettingsMode':
      return i18n._(msg`Game settings file`)
    default:
      return prefCopy(i18n, key).label
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
  const directPlay =
    resolveOverride('defaultLaunchMethod', launch, overrides) === 'direct' ||
    info?.directInstall === true
  const gameHours = useSettings((s) => gamePrefs(s, info?.id).saveBackupHours)
  const scheduleOff = Number(resolveOverride('saveBackupHours', String(gameHours), overrides)) === 0
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
    <>
      <Typography sx={{ fontSize: 'medium.fontSize', color: 'text.secondary' }}>
        {t`These apply only to this profile; Use default follows the game's settings.`}
      </Typography>
      <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
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
          const disabled = key === 'saveBackupKeep' && scheduleOff
          let hint = ''
          if (disabled) {
            hint = t`Enable scheduled save backups first.`
          } else if (key === 'updateModsBeforePlayDefault' && nexus && !premium) {
            hint = t`Free ${{ source: sourceLabel('nexus') }} accounts click each download on ${{ source: sourceLabel('nexus') }}; those updates do not block Play.`
          }
          return (
            <SelectField
              key={key}
              label={label}
              hint={hint || undefined}
              disabled={disabled}
              displayEmpty={true}
              value={choice.useGame ? '' : choice.value}
              onChange={(next) =>
                onChange(
                  applyRow(
                    overrides,
                    key,
                    next === '' ? { useGame: true } : { useGame: false, value: next },
                  ),
                )
              }
            >
              <MenuItem value="">{t`Use default`}</MenuItem>
              {choicesFor(key).map((o) => (
                <MenuItem key={o.value} value={o.value}>
                  {o.label}
                </MenuItem>
              ))}
            </SelectField>
          )
        })}
      </Box>
    </>
  )
}
