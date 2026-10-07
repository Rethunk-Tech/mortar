import { useLingui } from '@lingui/react/macro'
import { Box, Chip, Link, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { formatKb } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useNexus } from '../settings/nexus.ts'
import { Fold } from '../shell/Fold.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { type Block, parseBBCode, safeUrl } from './bbcode.ts'
import { DependencyChips } from './DependencyChips.tsx'
import { nexusDependencies } from './dependencies.ts'
import { openPage } from './menu.ts'
import { NewSinceLooked } from './NewSince.tsx'
import { NexusAccountActions } from './NexusAccountActions.tsx'
import {
  changelogIsNewSinceLooked,
  fileIsNewSinceLooked,
  useLookedSnapshot,
  useNexusEntry,
} from './nexusDetails.ts'
import { currentFiles, formatCount, isNewer, recentChangelogs } from './nexusFormat.ts'
import { goneCaption, nexusPageMark } from './nexusMark.ts'
import { heading } from './paper.ts'

const text = { fontSize: 13 } as const
const muted = { fontSize: 12, color: 'text.secondary' } as const
const noWrap = { whiteSpace: 'nowrap' } as const
const FILES_SHOWN = 5
const PANEL_COLUMNS = 2
const WIDE_COLUMNS = 3
const BOLD = 700

function Rich({ blocks }: { blocks: Block[] }) {
  return blocks.map((b) => (
    <Typography
      key={b.id}
      component="div"
      sx={{
        ...text,
        fontWeight: b.kind === 'heading' ? BOLD : undefined,
        mt: b.kind === 'heading' ? 1 : 0,
        pl: b.kind === 'item' ? 2 : 0,
        overflowWrap: 'anywhere',
      }}
    >
      {b.kind === 'item' ? '• ' : null}
      {b.runs.map((r) =>
        r.href ? (
          <Link
            key={r.id}
            component="button"
            onClick={() => (r.href ? openPage(r.href) : undefined)}
            sx={{ ...text, verticalAlign: 'baseline', textAlign: 'left' }}
          >
            {r.text}
          </Link>
        ) : (
          <Box
            key={r.id}
            component="span"
            sx={{
              fontWeight: r.bold ? BOLD : undefined,
              fontStyle: r.italic ? 'italic' : undefined,
            }}
          >
            {r.text}
          </Box>
        ),
      )}
    </Typography>
  ))
}

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Box sx={{ minWidth: 0 }}>
      <Typography sx={muted}>{label}</Typography>
      <Typography component="div" noWrap={true} sx={text}>
        {children}
      </Typography>
    </Box>
  )
}

function Facts({ details, mod, columns }: { details: Details; mod: Mod; columns: number }) {
  const { t, i18n } = useLingui()
  const { page, category } = details
  const uploader = safeUrl(page.uploaderUrl)
  const newer = isNewer(page.version, mod.version)
  return (
    <Box
      sx={{ display: 'grid', gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`, gap: 1.5 }}
    >
      <Fact label={t`Latest on Nexus`}>
        <Box component="span" sx={{ color: newer ? 'primary.main' : undefined }}>
          {page.version || '—'}
        </Box>
      </Fact>
      <Fact label={t`Category`}>{category || '—'}</Fact>
      <Fact label={t`Uploaded by`}>
        {uploader ? (
          <Link component="button" onClick={() => openPage(uploader)} sx={text}>
            {page.uploadedBy}
          </Link>
        ) : (
          page.uploadedBy || '—'
        )}
      </Fact>
      <Fact label={t`Endorsements`}>{formatCount(page.endorsements, i18n.locale)}</Fact>
      <Fact label={t`Downloads`}>
        {t`${formatCount(page.downloads, i18n.locale)} · ${formatCount(page.uniqueDownloads, i18n.locale)} unique`}
      </Fact>
      <Fact label={t`Created · updated`}>
        {`${formatWhen(page.created) || '—'} · ${formatWhen(page.updated) || '—'}`}
      </Fact>
    </Box>
  )
}

function Files({
  details,
  fileId,
  looked,
}: {
  details: Details
  fileId: number
  looked: { newestFileUnix: number; newestChange: string } | undefined
}) {
  const { t } = useLingui()
  const files = currentFiles(details.files ?? [], fileId)
  const installed = files[0]?.fileId === fileId ? files[0] : undefined
  const others = installed ? files.slice(1) : files
  const line = (f: (typeof files)[number]) =>
    [f.version, formatKb(f.sizeKb), formatWhen(f.uploaded)].filter(Boolean).join(' · ')
  return (
    <>
      {installed ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
          <Typography sx={heading}>{t`Installed file`}</Typography>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'baseline', flexWrap: 'wrap' }}>
            <Typography sx={{ ...text, overflowWrap: 'anywhere' }}>
              {installed.name || installed.fileName}
            </Typography>
            <NewSinceLooked show={fileIsNewSinceLooked(installed.uploaded, looked)} />
          </Box>
          <Typography sx={muted}>{line(installed)}</Typography>
          {installed.description ? (
            <Box sx={{ color: 'text.secondary' }}>
              <Rich blocks={parseBBCode(installed.description)} />
            </Box>
          ) : null}
        </Box>
      ) : null}
      {others.length > 0 ? (
        <Fold title={t`Current files on Nexus (${others.length})`}>
          {others.slice(0, FILES_SHOWN).map((f) => (
            <Box key={f.fileId} sx={{ display: 'flex', gap: 1, alignItems: 'baseline' }}>
              <Typography
                noWrap={true}
                title={f.name || f.fileName}
                sx={{ ...text, flex: 1, minWidth: 0 }}
              >
                {f.name || f.fileName}
              </Typography>
              <NewSinceLooked show={fileIsNewSinceLooked(f.uploaded, looked)} />
              <Typography sx={{ ...muted, ...noWrap }}>{line(f)}</Typography>
            </Box>
          ))}
          {others.length > FILES_SHOWN ? (
            <Typography
              sx={muted}
            >{t`And ${others.length - FILES_SHOWN} more on the Nexus page`}</Typography>
          ) : null}
        </Fold>
      ) : null}
    </>
  )
}

function Loaded({
  details,
  mod,
  fileId,
  modId,
  inPanel,
}: {
  details: Details
  mod: Mod
  fileId: number
  modId: number
  inPanel: boolean
}) {
  const { t } = useLingui()
  const { page } = details
  const logs = details.changelogs ?? []
  const description = page.description ? parseBBCode(page.description) : []
  const looked = useLookedSnapshot(modId, details)
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      {page.adult ||
      nexusPageMark(page.status, page.available, page.updated, page.created).kind !== '' ? (
        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
          {page.adult ? (
            <Chip size="small" color="warning" variant="outlined" label={t`Adult content`} />
          ) : null}
          {(() => {
            const mark = nexusPageMark(page.status, page.available, page.updated, page.created)
            const labelled = goneCaption(mark, {
              hidden: t`Hidden on Nexus`,
              hiddenDated: t`Hidden on Nexus · ${mark.date}`,
              removed: t`Removed from Nexus`,
              removedDated: t`Removed from Nexus · ${mark.date}`,
            })
            if (labelled === '') {
              return null
            }
            return (
              <Chip
                size="small"
                color="warning"
                variant="outlined"
                icon={<TriangleAlert size={14} aria-hidden={true} />}
                label={labelled}
              />
            )
          })()}
        </Box>
      ) : null}
      {page.summary && !inPanel ? (
        <Typography sx={{ ...text, color: 'text.secondary' }}>{page.summary}</Typography>
      ) : null}
      <Facts details={details} mod={mod} columns={inPanel ? PANEL_COLUMNS : WIDE_COLUMNS} />
      <NexusAccountActions modId={modId} version={mod.version} endorsement={page.endorsement} />
      {(page.requirements ?? []).length > 0 ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          <Typography sx={heading}>{t`Dependencies`}</Typography>
          <DependencyChips deps={nexusDependencies(page.requirements)} />
        </Box>
      ) : null}
      <Files details={details} fileId={fileId} looked={looked} />
      {logs.length > 0 ? (
        <Fold title={t`Changelog on Nexus`}>
          {recentChangelogs(logs).map((c) => (
            <Box key={c.version} sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'baseline' }}>
                <Typography sx={{ ...text, fontWeight: BOLD }}>{c.version}</Typography>
                <NewSinceLooked show={changelogIsNewSinceLooked(c.version, looked)} />
              </Box>
              <Typography sx={{ ...text, pl: 2, whiteSpace: 'pre-line', overflowWrap: 'anywhere' }}>
                {(c.notes ?? []).map((n) => `• ${n}`).join('\n')}
              </Typography>
            </Box>
          ))}
        </Fold>
      ) : null}
      {description.length > 0 ? (
        <Fold title={t`Description`}>
          <Rich blocks={description} />
        </Fold>
      ) : null}
    </Box>
  )
}

// What Nexus says about a Nexus-installed mod: served from Mortar's cache, which refreshes once a day while signed in.
// `inPanel` is the narrow details panel, which already shows the page's summary.
export function NexusDetails({
  mod,
  modId,
  fileId,
  inPanel = false,
}: {
  mod: Mod
  modId: number
  fileId: number
  inPanel?: boolean
}) {
  const { t } = useLingui()
  const signedIn = useNexus((s) => s.signedIn)
  const entry = useNexusEntry(modId)
  if (entry?.details) {
    return (
      <Loaded
        key={modId}
        details={entry.details}
        mod={mod}
        fileId={fileId}
        modId={modId}
        inPanel={inPanel}
      />
    )
  }
  if (!entry) {
    return <LoadingRow>{t`Reading the Nexus page…`}</LoadingRow>
  }
  return (
    <Typography sx={muted}>
      {signedIn
        ? t`Could not read the Nexus page: ${entry.error ?? ''}`
        : t`Sign in to Nexus Mods in Settings to see more from this mod's page.`}
    </Typography>
  )
}

export { Rich }
