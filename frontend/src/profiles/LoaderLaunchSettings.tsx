import type { MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { FormControlLabel, MenuItem, Switch } from '@mui/material'
import type { LaunchSetting } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loader/models.ts'
import {
  LoaderLaunchSettings as ReadSettings,
  SetLoaderLaunchSetting,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useGameInfo } from '../games/info.ts'
import { FormSection, SelectField } from '../shell/dialog/FormDialog.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportUnexpected } from '../toasts/report.ts'

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

const labelOf = (
  names: Record<string, MessageDescriptor>,
  key: string,
  say: (m: MessageDescriptor) => string,
) => {
  const m = names[key]
  return m ? say(m) : key
}

// The launch options the profile's loader keeps in the profile's own files, grouped under the loader's name; nothing
// shows for a loader that has none. Each change is written as it is made.
export function LoaderLaunchSettings({
  gameId,
  profileId,
  loader,
}: {
  gameId: string
  profileId: string
  loader: string
}) {
  const { t, i18n } = useLingui()
  const loaders = useGameInfo(gameId)?.loaders ?? []
  const name = (loaders.find((l) => l.id === loader) ?? loaders[0])?.name ?? ''
  const { data: settings, setData: setSettings } = useLoaded(
    () => ReadSettings(gameId, profileId).then((got) => got ?? NO_SETTINGS),
    [gameId, profileId],
    NO_SETTINGS,
    reportUnexpected,
  )
  if (settings.length === 0) {
    return null
  }
  const set = (id: string, value: string) => {
    SetLoaderLaunchSetting(gameId, profileId, id, value)
      .then(() => setSettings((all) => all.map((s) => (s.id === id ? { ...s, value } : s))))
      .catch(reportUnexpected)
  }
  return (
    <FormSection
      title={name}
      description={t`Saved in the profile's config folder and applied the next time the game starts.`}
    >
      {settings.map((s) => {
        const label = labelOf(LABELS, s.id, (m) => i18n._(m))
        if (s.choices?.length) {
          return (
            <SelectField key={s.id} label={label} value={s.value} onChange={(v) => set(s.id, v)}>
              {s.choices.map((c) => (
                <MenuItem key={c} value={c}>
                  {labelOf(CHOICES, c, (m) => i18n._(m))}
                </MenuItem>
              ))}
            </SelectField>
          )
        }
        return (
          <FormControlLabel
            key={s.id}
            sx={{ display: 'flex', m: 0, gap: 1 }}
            control={
              <Switch
                checked={s.value === 'true'}
                onChange={(event) => set(s.id, String(event.target.checked))}
              />
            }
            label={label}
          />
        )
      })}
    </FormSection>
  )
}
