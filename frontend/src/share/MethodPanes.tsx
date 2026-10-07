import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Copy, FileJson, Package, Save, Share2 } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import {
  ExportCode,
  ExportModpackDialog,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { SetLanSharing } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import type { Saved } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import {
  ExportCollection,
  SaveFile,
  ShowExportedCollection,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { listNames } from '../i18n/list.ts'
import { SendDialog } from '../lan/SendDialog.tsx'
import { useSettings } from '../settings/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { MONO } from '../theme/theme.ts'
import { reportError, reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { copyText } from './copyText.ts'
import { Included } from './Included.tsx'
import { ListFormat } from './ListFormat.tsx'
import { type MeterLevel, meter, type ShownInfo, suggestFile } from './logic.ts'
import { type ShareInclude, toShareInclude } from './shareDefaults.ts'

const PERCENT = 100

const note = { fontSize: 13, color: 'text.secondary', lineHeight: 1.5 } as const

function Pane({ children }: { children: ReactNode }) {
  return <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.25 }}>{children}</Box>
}

const LEVEL_COLOUR: Record<MeterLevel, string> = {
  ok: 'success.main',
  warn: 'warning.main',
  over: 'error.main',
}

function Meter({ info }: { info: ShownInfo }) {
  const { t, i18n } = useLingui()
  const { ratio, level } = meter(info.length, info.limit)
  const length = info.length.toLocaleString(i18n.locale)
  const limit = info.limit.toLocaleString(i18n.locale)
  const verdict = {
    ok: t`${length} of ${limit} characters · fits one Discord message`,
    warn: t`${length} of ${limit} characters · close to the Discord message limit`,
    over: t`${length} of ${limit} characters · too long for one Discord message`,
  }[level]
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
      <Box
        role="meter"
        aria-label={t`Link length`}
        aria-valuemin={0}
        aria-valuemax={info.limit}
        aria-valuenow={info.length}
        sx={{
          width: 96,
          height: 6,
          borderRadius: '3px',
          bgcolor: 'var(--mortar-hairline)',
          overflow: 'hidden',
        }}
      >
        <Box sx={{ width: `${ratio * PERCENT}%`, height: '100%', bgcolor: LEVEL_COLOUR[level] }} />
      </Box>
      <Typography sx={note}>{verdict}</Typography>
    </Box>
  )
}

interface IncludeProps {
  info: ShownInfo
  include: ShareInclude
  onInclude: (next: ShareInclude) => void
}

export function LinkPane({
  info,
  include,
  onInclude,
  onFile,
}: IncludeProps & { onFile: () => void }) {
  const { t } = useLingui()
  const large = suggestFile(info.count, info.tooLarge)
  return (
    <Pane>
      {info.tooLarge ? (
        <Typography sx={{ fontSize: 14, color: 'warning.light' }}>
          {t`This profile is too large for a link.`}
        </Typography>
      ) : (
        <>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Box
              sx={{
                flex: 1,
                minWidth: 0,
                display: 'flex',
                alignItems: 'center',
                height: 40,
                px: 1.5,
                bgcolor: 'var(--mortar-overlay-30)',
                borderRadius: '8px',
                fontFamily: MONO,
                fontSize: 13,
                userSelect: 'text',
              }}
            >
              <Typography noWrap={true} title={info.web} sx={{ font: 'inherit' }}>
                {info.web}
              </Typography>
            </Box>
            <Button
              variant="contained"
              startIcon={<Copy size={16} />}
              disabled={info.count === 0}
              onClick={() => copyText(info.web, t`Link copied`)}
              sx={{ height: 40, px: '18px', fontSize: 14, whiteSpace: 'nowrap' }}
            >
              {t`Copy link`}
            </Button>
          </Box>
          <Meter info={info} />
          <Typography sx={note}>
            {t`They see the profile name, the mod list and where each mod comes from.`}
          </Typography>
        </>
      )}
      {large ? (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Typography sx={{ flex: 1, fontSize: 14, color: 'warning.light' }}>
            {t`A .mortar file suits a profile this size better, and carries your mod settings too.`}
          </Typography>
          <Button variant="outlined" color="inherit" onClick={onFile}>
            {t`Use a file`}
          </Button>
        </Box>
      ) : null}
      <Included info={info} include={include} onInclude={onInclude} file={false} />
    </Pane>
  )
}

export function FilePane({
  info,
  game,
  profileId,
  keys,
  include,
  onInclude,
}: IncludeProps & { game: string; profileId: string; keys: string[] }) {
  const { t } = useLingui()
  const [saved, setSaved] = useState<Saved | null>(null)
  const [busy, run] = usePending()
  const save = () =>
    run(
      async () => {
        const result = await SaveFile(game, profileId, keys, toShareInclude(include))
        if (result.path) {
          setSaved(result)
          useToasts.getState().push({ kind: 'success', title: t`File saved` })
        }
      },
      { errorTitle: t`Could not save the file` },
    )
  const skipped = saved?.skipped ?? []
  return (
    <Pane>
      <Typography sx={note}>
        {t`A .mortar file holds the link's mods plus their settings files. Send it; they open it from Mortar's Import.`}
      </Typography>
      <Box>
        <Button
          variant="contained"
          startIcon={<Save size={16} />}
          disabled={busy || info.count === 0}
          onClick={save}
          sx={{ height: 40 }}
        >
          {t`Save…`}
        </Button>
      </Box>
      {saved?.path ? (
        <Typography
          sx={{ ...note, wordBreak: 'break-all' }}
        >{t`Saved to ${saved.path}`}</Typography>
      ) : null}
      {skipped.length > 0 ? (
        <Box>
          <Typography sx={{ fontSize: 13, fontWeight: 600 }}>
            {plural(skipped.length, {
              one: '# settings file was left out (too large or an unusual name):',
              other: '# settings files were left out (too large or an unusual name):',
            })}
          </Typography>
          <Box component="ul" sx={{ m: 0, pl: 2.5, ...note }}>
            {skipped.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </Box>
        </Box>
      ) : null}
      <Included info={info} include={include} onInclude={onInclude} file={true} />
    </Pane>
  )
}

export function ListPane({ info }: { info: ShownInfo }) {
  const { t } = useLingui()
  return (
    <Pane>
      <Typography sx={note}>
        {t`Copies the mod names as text. Pick a format; a long list is copied in parts that fit one message.`}
      </Typography>
      <Box>
        <ListFormat disabled={info.count === 0} />
      </Box>
    </Pane>
  )
}

export function NexusPane({ game, profileId }: { game: string; profileId: string }) {
  const { t } = useLingui()
  const exportDraft = async () => {
    const r = await ExportCollection(game, profileId)
    if (!r.path) {
      return
    }
    const skipped = r.skipped ?? []
    const next = t`Nexus expects a 7z. Repack collection.zip as a 7z with 7-Zip before uploading.`
    useToasts.getState().push({
      kind: skipped.length > 0 ? 'warning' : 'success',
      title: t`Collection draft saved`,
      body:
        skipped.length > 0
          ? `${next}\n${t`Left out: ${listNames(skipped, skipped.length)}`}`
          : next,
      action: {
        label: t`Show file`,
        run: () => ShowExportedCollection().catch(reportUnexpected),
      },
    })
  }
  return (
    <Pane>
      <Typography sx={note}>
        {t`Saves this profile as a collection draft to finish and upload on Nexus Mods.`}
      </Typography>
      <Box>
        <Button
          variant="contained"
          startIcon={<FileJson size={16} />}
          onClick={() => {
            exportDraft().catch(reportError(t`Could not export the collection draft`))
          }}
          sx={{ height: 40 }}
        >
          {t`Export as Nexus collection draft…`}
        </Button>
      </Box>
    </Pane>
  )
}

export function ThunderstorePane({ game, profileId }: { game: string; profileId: string }) {
  const { t } = useLingui()
  const [asking, setAsking] = useState(false)
  const publish = () => {
    setAsking(false)
    ExportCode(game, profileId, true)
      .then((code) =>
        useToasts.getState().push({
          kind: 'success',
          title: t`Code published`,
          body: code,
          action: { label: t`Copy code`, run: () => copyText(code, t`Code copied`) },
        }),
      )
      .catch((error: unknown) => toastError(t`Could not publish the code`, error))
  }
  const saveModpack = async () => {
    const r = await ExportModpackDialog(game, profileId, true)
    if (!r.path) {
      return
    }
    const left = [...(r.leftOut ?? []), ...(r.disabled ?? [])]
    useToasts.getState().push({
      kind: left.length > 0 ? 'warning' : 'success',
      title: t`Modpack saved`,
      body: left.length > 0 ? t`Left out: ${listNames(left)}` : r.path,
    })
  }
  return (
    <Pane>
      <Typography sx={note}>
        {t`For r2modman and other Thunderstore managers: a code anyone can enter, or a modpack zip with your config folder.`}
      </Typography>
      <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
        <Button
          variant="contained"
          startIcon={<Share2 size={16} />}
          onClick={() => setAsking(true)}
          sx={{ height: 40 }}
        >
          {t`Share as r2modman code…`}
        </Button>
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<Package size={16} />}
          onClick={() => {
            saveModpack().catch(reportError(t`Could not export the modpack`))
          }}
          sx={{ height: 40 }}
        >
          {t`Export as Thunderstore modpack…`}
        </Button>
      </Box>
      <ConfirmDialog
        open={asking}
        title={t`Share as r2modman code?`}
        body={t`This uploads the profile to Thunderstore and gives a public code.`}
        confirmLabel={t`Upload`}
        onCancel={() => setAsking(false)}
        onConfirm={publish}
      />
    </Pane>
  )
}

export function NearbyPane({ game, profileId }: { game: string; profileId: string }) {
  const { t } = useLingui()
  const lanSharing = useSettings((s) => s.lanSharing)
  const [open, setOpen] = useState(false)
  const [asking, setAsking] = useState(false)
  const [enabling, setEnabling] = useState(false)
  return (
    <Pane>
      <Typography sx={note}>
        {t`Send this profile to a computer on your network that runs Mortar. Paired computers get the whole file; the others get only the link.`}
      </Typography>
      <Box>
        <Button
          variant="contained"
          onClick={() => (lanSharing ? setOpen(true) : setAsking(true))}
          sx={{ height: 40 }}
        >
          {t`Choose a computer…`}
        </Button>
      </Box>
      <ConfirmDialog
        open={asking}
        title={t`Turn on sharing nearby?`}
        body={t`Other Mortar users on your local network will be able to find this computer and send you profiles. You can turn it off again in Settings › General.`}
        confirmLabel={t`Turn on and continue`}
        busy={enabling}
        onCancel={() => setAsking(false)}
        onConfirm={() => {
          setEnabling(true)
          SetLanSharing(true)
            .then(() => {
              setAsking(false)
              setOpen(true)
            })
            .catch(reportError(t`Could not turn on sharing nearby`))
            .finally(() => setEnabling(false))
        }}
      />
      <SendDialog open={open} game={game} profileId={profileId} onClose={() => setOpen(false)} />
    </Pane>
  )
}
