import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { FileJson, Package, Save, Share2 } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import {
  ExportCode,
  ExportR2zDialog,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { SetLanSharing } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import type { Saved } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import {
  ExportCollection,
  ShowExportedCollection,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { R2MODMAN } from '../brand/productNames.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { listNames } from '../i18n/list.ts'
import { SendPanel } from '../lan/SendPanel.tsx'
import { useSettings } from '../settings/store.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { Field } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { reportError, reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { copyText } from './copyText.ts'
import { Included } from './Included.tsx'
import { type MeterLevel, meter, type ShownInfo, suggestFile } from './logic.ts'
import { ModpackDialog } from './ModpackDialog.tsx'
import type { ShareInclude } from './shareDefaults.ts'
import { type Tile, TileGrid } from './TileGrid.tsx'

const PERCENT = 100

const note = { fontSize: 'base.fontSize', color: 'text.secondary', lineHeight: 1.5 } as const

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
  const verdict = [
    t`${length} of ${limit} characters`,
    {
      ok: t`fits one ${{ app: sourceLabel('discord') }} message`,
      warn: t`close to the ${{ app: sourceLabel('discord') }} message limit`,
      over: t`too long for one ${{ app: sourceLabel('discord') }} message`,
    }[level],
  ].join(' · ')
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
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
  total: number
  include: ShareInclude
  onInclude: (next: ShareInclude) => void
}

export function LinkPane({
  info,
  total,
  include,
  onInclude,
  onFile,
}: IncludeProps & { onFile: () => void }) {
  const { t } = useLingui()
  const large = suggestFile(info.count, info.tooLarge)
  return (
    <Pane>
      {info.tooLarge ? (
        <Typography sx={{ fontSize: 'medium.fontSize', color: 'warning.light' }}>
          {t`This profile is too large for a link.`}
        </Typography>
      ) : (
        <Field label={t`Link`} hint={<Meter info={info} />}>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              height: space.control,
              px: 1.5,
              bgcolor: 'var(--mortar-raised)',
              borderRadius: '6px',
              fontFamily: MONO,
              fontSize: 'base.fontSize',
              userSelect: 'text',
            }}
          >
            <Typography noWrap={true} title={info.web} sx={{ font: 'inherit' }}>
              {info.web}
            </Typography>
          </Box>
        </Field>
      )}
      {large ? (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
          <Typography sx={{ flex: 1, fontSize: 'medium.fontSize', color: 'warning.light' }}>
            {t`A .mortar file suits a profile this size better, and carries your mod settings too.`}
          </Typography>
          <Button variant="outlined" color="inherit" onClick={onFile}>
            {t`Use a file`}
          </Button>
        </Box>
      ) : null}
      <Included info={info} total={total} include={include} onInclude={onInclude} target="link" />
    </Pane>
  )
}

export function FilePane({
  info,
  total,
  include,
  onInclude,
  saved,
}: IncludeProps & { saved: Saved | null }) {
  const { t } = useLingui()
  const skipped = saved?.skipped ?? []
  return (
    <Pane>
      <Included info={info} total={total} include={include} onInclude={onInclude} target="file" />
      {saved?.path ? (
        <Typography
          sx={{ ...note, wordBreak: 'break-all' }}
        >{t`Saved to ${{ path: saved.path }}`}</Typography>
      ) : null}
      {skipped.length > 0 ? (
        <Box>
          <Typography sx={{ fontSize: 'base.fontSize', fontWeight: 600 }}>
            {t`${plural({ count: skipped.length }, { one: '# settings file was', other: '# settings files were' })} left out (too large or an unusual name):`}
          </Typography>
          <Box component="ul" sx={{ m: 0, pl: space.pad, ...note }}>
            {skipped.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </Box>
        </Box>
      ) : null}
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
    const next = t`${{ source: sourceLabel('nexus') }} expects a 7z. Repack collection.zip as a 7z with 7-Zip before uploading.`
    useToasts.getState().push({
      kind: skipped.length > 0 ? 'warning' : 'success',
      title: t`Collection draft saved`,
      body:
        skipped.length > 0
          ? `${next}\n${t`Left out: ${{ names: listNames(skipped, skipped.length) }}`}`
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
        {t`Saves this profile as a collection draft to finish and upload on ${{ source: sourceLabel('nexus') }}.`}
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
          {t`Export as ${{ source: sourceLabel('nexus') }} collection draft…`}
        </Button>
      </Box>
    </Pane>
  )
}

const TILE_ICON = 32

export function ThunderstorePane({ game, profileId }: { game: string; profileId: string }) {
  const { t } = useLingui()
  const [asking, setAsking] = useState(false)
  const [packing, setPacking] = useState(false)
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
  const saveR2z = async () => {
    const path = await ExportR2zDialog(game, profileId)
    if (!path) {
      return
    }
    useToasts.getState().push({
      kind: 'success',
      title: t`Saved .r2z`,
      body: path,
    })
  }
  const thunderstoreTiles: Tile<'code' | 'r2z' | 'modpack'>[] = [
    {
      id: 'code',
      icon: <Share2 size={TILE_ICON} />,
      name: t`${{ manager: R2MODMAN }} code`,
      hint: t`A code anyone can enter; uploads the profile`,
    },
    {
      id: 'r2z',
      icon: <Save size={TILE_ICON} />,
      name: t`.r2z file`,
      hint: t`A file to send, with your config files`,
    },
    {
      id: 'modpack',
      icon: <Package size={TILE_ICON} />,
      name: t`Modpack`,
      hint: t`A zip to publish on ${{ source: sourceLabel('thunderstore') }}`,
    },
  ]
  return (
    <Pane>
      <TileGrid
        label={t`How to share it`}
        lastUsed={null}
        padded={false}
        tiles={thunderstoreTiles}
        onChoose={(id) => {
          if (id === 'code') {
            setAsking(true)
          } else if (id === 'r2z') {
            saveR2z().catch(reportError(t`Could not save the file`))
          } else {
            setPacking(true)
          }
        }}
      />
      {packing ? (
        <ModpackDialog game={game} profileId={profileId} onClose={() => setPacking(false)} />
      ) : null}
      <ConfirmationDialog
        open={asking}
        title={t`Share as ${{ manager: R2MODMAN }} code?`}
        body={t`This uploads the profile to ${{ source: sourceLabel('thunderstore') }} and gives a public code.`}
        confirmLabel={t`Upload`}
        onCancel={() => setAsking(false)}
        onConfirm={publish}
      />
    </Pane>
  )
}

export function NearbyPane({
  game,
  profileId,
  info,
  total,
  include,
  onInclude,
}: {
  game: string
  profileId: string
  info: ShownInfo
  total: number
  include: ShareInclude
  onInclude: (next: ShareInclude) => void
}) {
  const { t } = useLingui()
  const lanSharing = useSettings((s) => s.lanSharing)
  const [enabling, setEnabling] = useState(false)
  if (lanSharing) {
    // A paired computer copies archives added from disk, so only the other mods the link cannot name are left out.
    const shown = { ...info, leftOut: info.leftOut.filter((o) => o.reason !== 'local') }
    return (
      <Pane>
        <Included
          info={shown}
          total={total}
          include={include}
          onInclude={onInclude}
          target="paired"
        />
        <SendPanel game={game} profileId={profileId} include={include} onSent={() => undefined} />
      </Pane>
    )
  }
  return (
    <Pane>
      <Typography
        sx={{ fontSize: 'lead.fontSize', fontWeight: 600 }}
      >{t`Enable sharing nearby?`}</Typography>
      <Typography sx={note}>
        {t`Other Mortar users on your local network will be able to find this computer and send you profiles. You can disable it again in Settings › General.`}
      </Typography>
      <Box>
        <Button
          variant="contained"
          disabled={enabling}
          onClick={() => {
            setEnabling(true)
            SetLanSharing(true)
              .catch(reportError(t`Could not enable sharing nearby`))
              .finally(() => setEnabling(false))
          }}
          sx={{ height: 40 }}
        >
          {t`Enable and continue`}
        </Button>
      </Box>
    </Pane>
  )
}
