import type { MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, FormControlLabel, MenuItem, Switch, TextField, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { LaunchSetting } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loader/models.ts'
import {
  LoaderLaunchSettings as ReadSettings,
  SetLoaderLaunchSetting,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useGameInfo } from '../games/info.ts'
import { reportUnexpected } from '../toasts/report.ts'

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
  const [settings, setSettings] = useState<LaunchSetting[]>([])
  useEffect(() => {
    ReadSettings(gameId, profileId)
      .then((got) => setSettings(got ?? []))
      .catch(reportUnexpected)
  }, [gameId, profileId])
  if (settings.length === 0) {
    return null
  }
  const set = (id: string, value: string) => {
    SetLoaderLaunchSetting(gameId, profileId, id, value)
      .then(() => setSettings((all) => all.map((s) => (s.id === id ? { ...s, value } : s))))
      .catch(reportUnexpected)
  }
  return (
    <Box sx={{ mt: 1 }}>
      <Typography sx={{ fontWeight: 600, mt: 1 }}>{name}</Typography>
      {settings.map((s) => {
        const label = labelOf(LABELS, s.id, (m) => i18n._(m))
        if (s.choices?.length) {
          return (
            <TextField
              key={s.id}
              select={true}
              fullWidth={true}
              margin="dense"
              label={label}
              value={s.value}
              onChange={(event) => set(s.id, event.target.value)}
            >
              {s.choices.map((c) => (
                <MenuItem key={c} value={c}>
                  {labelOf(CHOICES, c, (m) => i18n._(m))}
                </MenuItem>
              ))}
            </TextField>
          )
        }
        return (
          <FormControlLabel
            key={s.id}
            sx={{ display: 'flex' }}
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
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
        {t`Saved in the profile's config folder and applied the next time the game starts.`}
      </Typography>
    </Box>
  )
}
