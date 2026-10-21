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
import { Clipboard } from '@wailsio/runtime'
import { TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import { ReportTarget } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import {
  RunLines,
  RunLog,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { OpenWeb } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import type {
  Mod,
  Profile,
  Source,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  Log,
  Upload,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { anonymize } from '../console/anonymize.ts'
import { shareLogConfirm } from '../console/shareLog.ts'
import { gameName } from '../games/info.ts'
import { useLoader } from '../loader/store.ts'
import { useProfileLoader } from '../profiles/store.ts'
import { useMortarUpdate } from '../settings/updates.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { entryOf, nexusIdOf } from './lookup.ts'
import { nexusDomain } from './nexusDomain.ts'
import { buildModReport, type ModReportFields, modErrorLines } from './reportToAuthor.ts'

const REPORT_SOURCES = ['thunderstore', 'curseforge', 'vsmoddb']

const buttonSx = { whiteSpace: 'nowrap' } as const

interface ReportFieldsInput {
  game: string
  profile: Profile
  mod: Mod
  modLogName: string
  issueTitle: string
  entries: Awaited<ReturnType<typeof RunLines>>
  loader: { gameVersion: string; version: string } | null
  mortarVersion: string
  logShareURL: string
  target: { repo: string; url: string }
}

function reportFields(input: ReportFieldsInput): ModReportFields {
  const {
    game,
    profile,
    mod,
    modLogName,
    issueTitle,
    entries,
    loader,
    mortarVersion,
    logShareURL,
    target,
  } = input
  const source = entryOf(profile, mod.key)?.source
  return {
    modName: mod.name,
    modVersion: mod.version,
    gameName: gameName(game),
    gameVersion: loader?.gameVersion ?? '',
    smapiVersion: loader?.version ?? '',
    mortarVersion,
    errorLines: entries ? modErrorLines(entries, modLogName) : [],
    logShareURL,
    source,
    nexusDomain: nexusDomain(),
    githubRepo: source?.kind === 'github' ? (source.repo ?? '') : target.repo,
    nexusModId: nexusIdOf(profile, mod),
    pageUrl: target.url,
    issueTitle,
  }
}

// Where the mod's own site takes reports, for the sites Mortar installs from that are neither GitHub nor Nexus; empty
// when the site cannot say, so the report is still copied.
async function reportTarget(
  game: string,
  source: Source | null | undefined,
): Promise<{ repo: string; url: string }> {
  const none = { repo: '', url: '' }
  if (!(source && REPORT_SOURCES.includes(source.kind))) {
    return none
  }
  try {
    return (await ReportTarget(game, source.kind, source.name)) ?? none
  } catch {
    return none
  }
}

async function finishReport(opts: {
  i18n: I18n
  copiedTitle: string
  pasteHint: string
  cutHint: string
  fields: ModReportFields
  push: ReturnType<typeof useToasts.getState>['push']
}): Promise<void> {
  const { i18n, copiedTitle, pasteHint, cutHint, fields, push } = opts
  const report = buildModReport(i18n, fields)
  await Clipboard.SetText(report.text)
  if (report.url !== '') {
    await OpenWeb(report.url)
  }
  if (report.nexus || report.page || report.cut) {
    push({
      kind: 'success',
      title: copiedTitle,
      body: report.cut ? cutHint : pasteHint,
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
  // smapi.io's parser reads only SMAPI's log; other loaders' reports go out without a log link.
  const profileLoader = useProfileLoader(profile.id)
  const shares = profileLoader?.share === true
  const start = () => {
    if (!shares) {
      skipShare()
      return
    }
    setOpen(true)
    setLog(null)
    const read = runId ? RunLog(game, profile.id, runId) : Log(game, profile.id)
    read.then(
      (text) => setLog(text ?? ''),
      (e: unknown) => {
        toastError(t`Could not read the game log`, e)
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
      const target = await reportTarget(game, entryOf(profile, mod.key)?.source)
      const fields = reportFields({
        game,
        profile,
        mod,
        modLogName,
        issueTitle: t`Error in ${mod.name}`,
        entries,
        loader,
        mortarVersion,
        logShareURL,
        target,
      })
      await finishReport({
        i18n,
        copiedTitle: t`Report copied`,
        pasteHint: t`Paste it on the mod's page.`,
        cutHint: t`The issue holds the start of it. Paste the whole report over it.`,
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
      const url = await Upload(game, profileLoader?.id ?? '', hideUserName ? anonymize(log) : log)
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
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
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
        <DialogActions sx={{ px: space.pad, pb: space.pad, flexWrap: 'wrap', gap: space.gap }}>
          <Button disabled={uploading} onClick={close} sx={buttonSx}>
            {t`Cancel`}
          </Button>
          <Button disabled={uploading || log === null} onClick={skipShare} sx={buttonSx}>
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
