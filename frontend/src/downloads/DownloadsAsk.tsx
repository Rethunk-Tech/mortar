import { useLingui } from '@lingui/react/macro'
import { Button, Snackbar, Stack, Typography } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { useEffect, useState } from 'react'
import type { Arrival } from '../../bindings/github.com/Rethunk-AI/mortar/internal/dlwatch/models.ts'
import {
  Ignore,
  Install,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/dlwatch/service.ts'
import { useFomod } from '../fomod/store.ts'
import { useInstall } from '../install/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'

export function DownloadsAsk() {
  const { t } = useLingui()
  const [ask, setAsk] = useState<Arrival | null>(null)
  useEffect(() => {
    Events.On('dlwatch:arrived', (event: { data: Arrival }) => {
      setAsk(event.data)
    })
  }, [])
  if (!ask) {
    return null
  }
  const close = () => setAsk(null)
  const profile = ask.profileName || t`the open profile`
  return (
    <Snackbar
      open={true}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      message={
        <Typography component="span" sx={{ fontSize: 14 }}>
          {t`Install ${ask.name} into ${profile}?`}
        </Typography>
      }
      action={
        <Stack direction="row" spacing={1}>
          <Button
            color="inherit"
            size="small"
            onClick={() => {
              Ignore(ask.path).catch(reportUnexpected)
              close()
            }}
          >
            {t`Ignore`}
          </Button>
          <Button
            color="inherit"
            size="small"
            onClick={() => {
              const { n, name } = ask
              close()
              Install(n)
                .then((res) => {
                  if (res.fomod) {
                    useFomod.getState().open({
                      game: ask.game,
                      profileId: ask.profileId,
                      key: res.fomod.key,
                      source: res.fomod.source,
                      ask: res.fomod,
                    })
                    return
                  }
                  if (res.remap) {
                    useInstall.getState().openRemap({
                      game: ask.game,
                      profileName: ask.profileName,
                      profileId: ask.profileId,
                      key: res.remap.key,
                      source: res.remap.source,
                      ask: res.remap,
                    })
                    return
                  }
                  if (res.profile) {
                    useProfiles.getState().replace(res.profile)
                  }
                })
                .catch((e: unknown) => {
                  toastError(t`Could not install ${name}`, e)
                })
            }}
          >
            {t`Install`}
          </Button>
        </Stack>
      }
    />
  )
}
