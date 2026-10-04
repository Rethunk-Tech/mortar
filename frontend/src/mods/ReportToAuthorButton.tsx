import type { I18n } from '@lingui/core'
import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Switch,
  Typography,
} from '@mui/material'
import { Browser, Clipboard } from '@wailsio/runtime'
import { TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import {
  RunLines,
  RunLog,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  Log,
  Upload,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { anonymize } from '../console/anonymize.ts'
import { shareLogConfirm } from '../console/shareLog.ts'
import { useLoader } from '../loader/store.ts'
import { useMortarUpdate } from '../settings/updates.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { entryOf, nexusIdOf } from './lookup.ts'
import { buildModReport, type ModReportFields, modErrorLines } from './reportToAuthor.ts'

const buttonSx = { whiteSpace: 'nowrap' } as const

interface ReportFieldsInput {
  profile: Profile
  mod: Mod
  modLogName: string
  issueTitle: string
  entries: Awaited<ReturnType<typeof RunLines>>
  loader: { gameVersion: string; version: string } | null
  mortarVersion: string
  logShareURL: string
}

function reportFields(input: ReportFieldsInput): ModReportFields {
  const { profile, mod, modLogName, issueTitle, entries, loader, mortarVersion, logShareURL } =
    input
  const source = entryOf(profile, mod.key)?.source
  return {
    modName: mod.name,
    modVersion: mod.version,
    gameVersion: loader?.gameVersion ?? '',
    smapiVersion: loader?.version ?? '',
    mortarVersion,
    errorLines: entries ? modErrorLines(entries, modLogName) : [],
    logShareURL,
    source,
    nexusDomain: 'stardewvalley',
    githubRepo: source?.kind === 'github' ? (source.repo ?? '') : '',
    nexusModId: nexusIdOf(profile, mod),
    issueTitle,
  }
}

async function finishReport(opts: {
  i18n: I18n
  copiedTitle: string
  nexusHint: string
  fields: ModReportFields
  push: ReturnType<typeof useToasts.getState>['push']
}): Promise<void> {
  const { i18n, copiedTitle, nexusHint, fields, push } = opts
  const report = buildModReport(i18n, fields)
  await Clipboard.SetText(report.text)
  if (report.url !== '') {
    await Browser.OpenURL(report.url)
  }
  if (report.nexus) {
    push({
      kind: 'success',
      title: copiedTitle,
      body: nexusHint,
    })
    return
  }
  push({ kind: 'success', title: copiedTitle })
}

export function ReportToAuthorButton({
  game,
  profile,
  runId,
  mod,
  modLogName,
  size = 'small',
  variant = 'outlined',
}: {
  game: string
  profile: Profile
  runId: string
  mod: Mod
  modLogName: string
  size?: 'small' | 'medium'
  variant?: 'outlined' | 'text'
}) {
  const { t, i18n } = useLingui()
  const push = useToasts((s) => s.push)
  const loader = useLoader((s) => s.status)
  const mortarVersion = useMortarUpdate((s) => s.info?.version ?? '')
  const [open, setOpen] = useState(false)
  const [log, setLog] = useState<string | null>(null)
  const [hideUserName, setHideUserName] = useState(true)
  const [uploading, setUploading] = useState(false)
  const start = () => {
    setOpen(true)
    setLog(null)
    const read = runId ? RunLog(game, profile.id, runId) : Log(game, profile.id)
    read.then(
      (text) => setLog(text ?? ''),
      (e: unknown) => {
        toastError(t`Could not read the SMAPI log`, e)
        setOpen(false)
      },
    )
  }
  const close = () => {
    if (!uploading) {
      setOpen(false)
    }
  }
  const runWithLink = async (logShareURL: string) => {
    try {
      const entries = runId ? await RunLines(game, profile.id, runId) : null
      const fields = reportFields({
        profile,
        mod,
        modLogName,
        issueTitle: t`Error in ${mod.name}`,
        entries,
        loader,
        mortarVersion,
        logShareURL,
      })
      await finishReport({
        i18n,
        copiedTitle: t`Report copied`,
        nexusHint: t`Paste it on the mod's bug tracker.`,
        fields,
        push,
      })
      setOpen(false)
    } catch (e: unknown) {
      reportUnexpected(e)
    }
  }
  const skipShare = () => {
    setUploading(true)
    runWithLink('')
      .catch(reportUnexpected)
      .finally(() => setUploading(false))
  }
  const upload = async () => {
    if (log === null || log === '') {
      await skipShare()
      return
    }
    setUploading(true)
    try {
      const url = await Upload(hideUserName ? anonymize(log) : log)
      await runWithLink(url)
    } catch (e: unknown) {
      toastError(t`Could not upload the log`, e)
    } finally {
      setUploading(false)
    }
  }
  const bytes = new TextEncoder().encode(log ?? '').length
  return (
    <>
      <Button size={size} variant={variant} onClick={start} sx={{ height: 28 }}>
        {t`Report to author…`}
      </Button>
      <Dialog
        open={open}
        onClose={close}
        slotProps={{ paper: { sx: { width: 520, maxWidth: 'calc(100% - 32px)' } } }}
      >
        <DialogTitle sx={{ fontSize: 22, fontWeight: 700 }}>{t`Report to author`}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {log === null ? (
            <Typography sx={{ fontSize: 14 }}>{t`Reading the log…`}</Typography>
          ) : (
            <>
              <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
                {shareLogConfirm(i18n, bytes)}
              </Typography>
              <Alert severity="warning" icon={<TriangleAlert size={16} aria-hidden={true} />}>
                {t`The log holds folder paths from this computer, which can include your user name. Anyone with the link can read it.`}
              </Alert>
              <FormControlLabel
                control={
                  <Switch
                    checked={hideUserName}
                    onChange={(_, value) => setHideUserName(value)}
                    disabled={log === ''}
                  />
                }
                label={t`Hide my user name`}
              />
            </>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5, flexWrap: 'wrap', gap: 1 }}>
          <Button variant="outlined" disabled={uploading} onClick={close} sx={buttonSx}>
            {t`Cancel`}
          </Button>
          <Button
            variant="outlined"
            disabled={uploading || log === null}
            onClick={skipShare}
            sx={buttonSx}
          >
            {t`Skip sharing`}
          </Button>
          <Button
            variant="contained"
            disabled={uploading || log === null || log === ''}
            onClick={upload}
            sx={buttonSx}
          >
            {uploading ? t`Uploading…` : t`Share log and continue`}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
