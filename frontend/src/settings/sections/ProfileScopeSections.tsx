import type { MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { useEffect, useRef, useState } from 'react'
import {
  GameSettings as GetGameSettings,
  SetGameSettings,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type { LaunchSetting } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/loader/models.ts'
import {
  LoaderLaunchSettings as ReadSettings,
  SetLoaderLaunchSetting,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useGameInfo } from '../../games/info.ts'
import { formSettingsFromBackend } from '../../profiles/formSettingsFromBackend.ts'
import { GameSettings, type GameSettingsValues } from '../../profiles/GameSettings.tsx'
import { useLoaded } from '../../shell/useLoaded.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { PrefSelect, PrefSwitch } from '../PrefControls.tsx'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'

const NO_SETTINGS: LaunchSetting[] = []

const LABELS: Record<string, MessageDescriptor> = {
  console: msg`Show the console window`,
  logLevel: msg`Log level`,
}

const CHOICES: Record<string, MessageDescriptor> = {
  quiet: msg`Warnings and errors`,
  default: msg`Standard`,
  debug: msg`Debug`,
  all: msg`Everything`,
}

// Typing in a number field saves once it pauses, not on every key.
const SAVE_PAUSE_MS = 500

/** The launch options the profile's loader keeps in the profile's own files (BepInEx.cfg); each change is written as
 * it is made. Nothing shows for a loader that has none. */
// Both sections show only on the open profile's scope: profileId is empty otherwise.

export function LoaderProfileSettings({
  game,
  profileId,
  loaderName,
}: {
  game: string
  profileId: string
  loaderName: string
}) {
  const { t, i18n } = useLingui()
  const { data: settings, setData: setSettings } = useLoaded(
    profileId ? () => ReadSettings(game, profileId).then((got) => got ?? NO_SETTINGS) : null,
    [game, profileId],
    NO_SETTINGS,
    reportUnexpected,
  )
  if (settings.length === 0) {
    return null
  }
  const say = (names: Record<string, MessageDescriptor>, key: string) => {
    const m = names[key]
    return m ? i18n._(m) : key
  }
  const set = (id: string, value: string) => {
    SetLoaderLaunchSetting(game, profileId, id, value)
      .then(() => setSettings((all) => all.map((s) => (s.id === id ? { ...s, value } : s))))
      .catch(reportUnexpected)
  }
  return (
    <SettingsSection
      title={t`${{ name: loaderName }} in this profile`}
      description={t`Saved in the profile's config folder and applied the next time the game starts.`}
    >
      {settings.map((s) => {
        const label = say(LABELS, s.id)
        return (
          <SettingRow key={s.id} label={label}>
            {s.choices?.length ? (
              <PrefSelect
                value={s.value}
                options={s.choices.map((c) => ({ value: c, label: say(CHOICES, c) }))}
                label={label}
                onChange={(v) => set(s.id, v)}
              />
            ) : (
              <PrefSwitch
                checked={s.value === 'true'}
                label={label}
                onChange={(on) => set(s.id, String(on))}
              />
            )}
          </SettingRow>
        )
      })}
    </SettingsSection>
  )
}

/** The game's own options this profile starts it with (window, resolution, volume), saved as they change. */
export function ProfileGameOptions({ game, profileId }: { game: string; profileId: string }) {
  // Only a game with a startup preferences file applies them; elsewhere the fields would change nothing.
  const startup = useGameInfo(game)?.startupSettings === true
  const { data } = useLoaded<GameSettingsValues | null>(
    startup && profileId
      ? () => GetGameSettings(game, profileId).then(formSettingsFromBackend)
      : null,
    [game, profileId],
    null,
    reportUnexpected,
  )
  const [value, setValue] = useState<GameSettingsValues | null>(null)
  useEffect(() => setValue(data), [data])
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  useEffect(() => () => clearTimeout(timer.current), [])
  const change = (next: GameSettingsValues | null) => {
    setValue(next)
    clearTimeout(timer.current)
    timer.current = setTimeout(() => {
      SetGameSettings(game, profileId, next ?? {}).catch(reportUnexpected)
    }, SAVE_PAUSE_MS)
  }
  if (!(startup && profileId)) {
    return null
  }
  return (
    <Box sx={{ px: 0.5 }}>
      <GameSettings profileId={profileId} value={value} onChange={change} />
    </Box>
  )
}
