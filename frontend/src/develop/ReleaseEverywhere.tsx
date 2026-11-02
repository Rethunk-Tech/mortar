import { useLingui } from '@lingui/react/macro'
import { Box, Button, CircularProgress, Typography } from '@mui/material'
import { CircleAlert, CircleCheck, Clock, Rocket } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { UploadSite } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { openWeb } from '../shell/openWeb.ts'
import { space } from '../theme/density.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { reportError } from '../toasts/report.ts'
import { packProject } from './builds.ts'
import { siteLabel, uploadResult } from './uploadSites.ts'

type State =
  | { kind: 'skipped'; reason: string }
  | { kind: 'waiting' }
  | { kind: 'uploading' }
  | { kind: 'done'; url: string }
  | { kind: 'failed'; error: string }

function DoneCell({ url }: { url: string }) {
  const { t } = useLingui()
  if (!url) {
    return (
      <Typography variant="caption" color="text.secondary">
        {t`Done`}
      </Typography>
    )
  }
  return (
    <Button
      size="small"
      onClick={() => openWeb(url).catch(reportError(t`Could not open the page`))}
    >
      {t`Open page`}
    </Button>
  )
}

/** Packs the project once, then uploads to each ready site in turn and lists every site's state. The sites run one
 * after another: each upload reads the same package file, and a log that fills in order is easier to follow than four
 * bars racing. One failure never stops the others, and a failed site has its own Retry. */
export function ReleaseEverywhere({
  game,
  dir,
  name,
  version,
  sites,
  onClose,
}: {
  game: string
  dir: string
  name: string
  version: string
  sites: UploadSite[]
  onClose: () => void
}) {
  const { t } = useLingui()
  const reason = (s: UploadSite) =>
    s.id === 'github' ? t`Not signed in with the gh CLI` : t`No credentials yet`
  const [states, setStates] = useState<Record<string, State>>(() =>
    Object.fromEntries(
      sites.map((s) => [
        s.id,
        s.ready
          ? { kind: 'waiting' }
          : { kind: 'skipped', reason: s.configured ? reason(s) : t`Not set up for this project` },
      ]),
    ),
  )
  const [packing, setPacking] = useState(true)
  const [packFailed, setPackFailed] = useState(false)
  const started = useRef<boolean | null>(null)
  const send = useCallback(
    async (id: string) => {
      const set = (state: State) => setStates((m) => ({ ...m, [id]: state }))
      set({ kind: 'uploading' })
      try {
        set({ kind: 'done', url: await uploadResult(id, { game, dir, version }) })
      } catch (e) {
        set({ kind: 'failed', error: errorDetails(e) })
      }
    },
    [game, dir, version],
  )
  useEffect(() => {
    if (started.current) {
      return
    }
    started.current = true
    ;(async () => {
      const pack = await packProject(game, dir)
      setPacking(false)
      // The project card shows why a package was refused.
      if (!pack?.zip || (pack.issues ?? []).length > 0) {
        setPackFailed(true)
        return
      }
      for (const s of sites.filter((x) => x.ready)) {
        await send(s.id)
      }
    })().catch(reportError(t`Could not release the mod`))
  }, [game, dir, sites, send, t])
  const running = packing || Object.values(states).some((s) => s.kind === 'uploading')
  return (
    <AppDialog
      open={true}
      onClose={onClose}
      title={t`Release ${{ name }} ${{ version }}`}
      icon={
        <ToneIcon tone="neutral">
          <Rocket size={22} />
        </ToneIcon>
      }
      size="sm"
      footer={
        <DialogFooter>
          <Button variant="contained" onClick={onClose} disabled={running}>
            {t`Close`}
          </Button>
        </DialogFooter>
      }
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
        {packing ? (
          <Typography color="text.secondary">{t`Building the package…`}</Typography>
        ) : null}
        {packFailed ? (
          <Typography color="error">{t`The package was refused. The project card says why.`}</Typography>
        ) : null}
        {sites.map((s) => {
          const st = states[s.id]
          if (!st) {
            return null
          }
          return (
            <Box
              key={s.id}
              sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minHeight: 36 }}
            >
              <Box sx={{ width: 20, display: 'grid', placeItems: 'center' }}>
                {st.kind === 'uploading' ? <CircularProgress size={16} /> : null}
                {st.kind === 'done' ? <CircleCheck size={18} /> : null}
                {st.kind === 'failed' ? <CircleAlert size={18} /> : null}
                {st.kind === 'waiting' || st.kind === 'skipped' ? <Clock size={18} /> : null}
              </Box>
              <Typography sx={{ flex: 1 }}>{siteLabel(s.id)}</Typography>
              {st.kind === 'waiting' ? (
                <Typography variant="caption" color="text.secondary">{t`Waiting`}</Typography>
              ) : null}
              {st.kind === 'uploading' ? (
                <Typography variant="caption" color="text.secondary">{t`Uploading…`}</Typography>
              ) : null}
              {st.kind === 'skipped' ? (
                <Typography variant="caption" color="text.secondary">
                  {t`Skipped: ${{ reason: st.reason }}`}
                </Typography>
              ) : null}
              {st.kind === 'done' ? <DoneCell url={st.url} /> : null}
              {st.kind === 'failed' ? (
                <>
                  <Typography
                    variant="caption"
                    color="error"
                    title={st.error}
                    noWrap={true}
                    sx={{ maxWidth: 200 }}
                  >
                    {st.error}
                  </Typography>
                  <Button
                    size="small"
                    disabled={running}
                    onClick={() => send(s.id).catch(reportError(t`Could not upload`))}
                  >
                    {t`Retry`}
                  </Button>
                </>
              ) : null}
            </Box>
          )
        })}
      </Box>
    </AppDialog>
  )
}
