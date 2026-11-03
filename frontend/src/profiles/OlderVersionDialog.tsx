import { useLingui } from '@lingui/react/macro'
import { Alert, Button, Link, MenuItem, Typography } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { useEffect, useState } from 'react'
import { OpenSteamConsole } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import {
  AdoptOlderVersion,
  DownloadDepotCommand,
  OlderVersionDepots,
  OlderVersionState,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { STEAM } from '../brand/productNames.ts'
import { copyText } from '../share/copyText.ts'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { SelectField, TextInput } from '../shell/dialog/FormDialog.tsx'
import { openWeb } from '../shell/openWeb.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { useProfiles } from './store.ts'

const POLL_MS = 2000

// Steam's console downloads the depot; Mortar only prepares the command and then watches the folder Steam fills.
export function OlderVersionDialog({
  gameId,
  profileId,
  open,
  onClose,
}: {
  gameId: string
  profileId: string
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const [busy, runPending] = usePending()
  const { data } = useLoaded(open ? () => OlderVersionDepots(gameId) : null, [open, gameId], null)
  const [depot, setDepot] = useState('')
  const [manifest, setManifest] = useState('')
  const [watching, setWatching] = useState(false)
  const [ready, setReady] = useState(false)
  const depotId = depot || data?.depots?.[0]?.id || ''
  const wanted = manifest.trim()
  const builds = data?.builds ?? []
  const pickBuild = (key: string) => {
    const b = builds.find((x) => `${x.depotId}:${x.manifest}` === key)
    if (b) {
      setDepot(b.depotId)
      setManifest(b.manifest)
    }
  }

  useEffect(() => {
    if (!(open && watching) || ready) {
      return
    }
    const timer = setInterval(() => {
      OlderVersionState(gameId, depotId)
        .then((s) => setReady(s?.ready ?? false))
        .catch(reportUnexpected)
    }, POLL_MS)
    return () => clearInterval(timer)
  }, [open, watching, ready, gameId, depotId])

  const close = () => {
    setWatching(false)
    setReady(false)
    onClose()
  }
  const start = () => {
    DownloadDepotCommand(gameId, depotId, wanted)
      .then((command) => {
        copyText(command, t`Command copied`)
        return OpenSteamConsole()
      })
      .then(() => setWatching(true))
      .catch(reportError(t`Could not open the ${{ launcher: STEAM }} console`))
  }
  const adopt = () =>
    runPending(
      () =>
        AdoptOlderVersion(gameId, profileId, depotId, wanted).then((p) => {
          useProfiles.getState().replace(p)
          close()
        }),
      { errorTitle: t`Could not copy the older version` },
    )

  let footer = (
    <Button variant="contained" disabled={depotId === '' || wanted === ''} onClick={start}>
      {t`Open the ${{ launcher: STEAM }} console`}
    </Button>
  )
  if (watching) {
    footer = (
      <Button variant="contained" disabled={!ready || busy} onClick={adopt}>
        {busy ? t`Copying…` : t`Use this version`}
      </Button>
    )
  }
  return (
    <AppDialog
      open={open}
      onClose={busy ? () => undefined : close}
      title={t`Get an older version`}
      footer={
        <DialogFooter>
          <Button onClick={close} disabled={busy}>
            {watching ? t`Cancel` : t`Close`}
          </Button>
          {footer}
        </DialogFooter>
      }
    >
      <div className="app-dialog-form">
        {watching ? (
          <>
            <Typography>
              {t`The command is on your clipboard. Paste it into the ${{ launcher: STEAM }} console and press Enter. Mortar waits for the download to finish.`}
            </Typography>
            <Typography sx={{ color: 'text.secondary' }}>
              {ready ? t`The download is done.` : t`Waiting for ${{ source: STEAM }}…`}
            </Typography>
          </>
        ) : (
          <>
            <SelectField label={t`Depot`} value={depotId} onChange={setDepot}>
              {(data?.depots ?? []).map((d) => (
                <MenuItem key={d.id} value={d.id}>
                  {d.id}
                </MenuItem>
              ))}
            </SelectField>
            {builds.length > 0 && (
              <SelectField
                label={t`Builds Mortar has seen`}
                value={
                  builds.some((b) => b.depotId === depotId && b.manifest === wanted)
                    ? `${depotId}:${wanted}`
                    : ''
                }
                onChange={pickBuild}
              >
                {builds.map((b) => (
                  <MenuItem key={`${b.depotId}:${b.manifest}`} value={`${b.depotId}:${b.manifest}`}>
                    {[
                      b.seen,
                      b.buildId && t`build ${{ commit: b.buildId }}`,
                      b.manifest,
                      b.current && t`current`,
                    ]
                      .filter(Boolean)
                      .join(' · ')}
                  </MenuItem>
                ))}
              </SelectField>
            )}
            <TextInput
              label={t`Manifest id of the version`}
              value={manifest}
              onChange={setManifest}
              placeholder="7081337227611034738"
              hint={
                <Link
                  component="button"
                  type="button"
                  onClick={() => {
                    openWeb(`https://steamdb.info/depot/${depotId}/manifests/`).catch(
                      reportError(t`Could not open the browser`),
                    )
                  }}
                >
                  {t`Look up manifests on SteamDB`}
                </Link>
              }
            />
            <Alert severity="info">
              {t`The ${{ launcher: STEAM }} console downloads the version. Mortar never asks for your ${{ launcher: STEAM }} login.`}
            </Alert>
          </>
        )}
      </div>
    </AppDialog>
  )
}
