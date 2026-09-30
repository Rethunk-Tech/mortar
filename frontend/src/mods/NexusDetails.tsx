import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Collapse, Link, Typography } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { ChevronDown, ChevronRight, TriangleAlert } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Details } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { type Block, parseBBCode, safeUrl } from './bbcode.ts'
import { NexusAccountActions } from './NexusAccountActions.tsx'
import { useNexusEntry } from './nexusDetails.ts'
import {
  currentFiles,
  formatCount,
  formatDate,
  formatSize,
  isNewer,
  recentChangelogs,
} from './nexusFormat.ts'
import { heading } from './paper.ts'

const text = { fontSize: 13 } as const
const muted = { fontSize: 12, color: 'text.secondary' } as const
const noWrap = { whiteSpace: 'nowrap' } as const
const FILES_SHOWN = 5
const BOLD = 700

const open = (url: string) => Browser.OpenURL(url).catch(reportUnexpected)

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
            onClick={() => (r.href ? open(r.href) : undefined)}
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

function Fold({ title, children }: { title: string; children: ReactNode }) {
  const [shown, setShown] = useState(false)
  const Icon = shown ? ChevronDown : ChevronRight
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Button
        size="small"
        onClick={() => setShown(!shown)}
        startIcon={<Icon size={14} aria-hidden={true} />}
        aria-expanded={shown}
        sx={{
          ...noWrap,
          ...muted,
          fontWeight: BOLD,
          textTransform: 'none',
          alignSelf: 'flex-start',
          px: 0.5,
        }}
      >
        {title}
      </Button>
      <Collapse in={shown} unmountOnExit={true}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, pl: 1 }}>{children}</Box>
      </Collapse>
    </Box>
  )
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

function Facts({ details, mod }: { details: Details; mod: Mod }) {
  const { t, i18n } = useLingui()
  const { page, category } = details
  const uploader = safeUrl(page.uploaderUrl)
  const newer = isNewer(page.version, mod.version)
  return (
    <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 1.5 }}>
      <Fact label={t`Latest on Nexus`}>
        <Box component="span" sx={{ color: newer ? 'primary.main' : undefined }}>
          {page.version || '—'}
        </Box>
      </Fact>
      <Fact label={t`Category`}>{category || '—'}</Fact>
      <Fact label={t`Uploaded by`}>
        {uploader ? (
          <Link component="button" onClick={() => open(uploader)} sx={text}>
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
        {`${formatDate(page.created, i18n.locale) || '—'} · ${formatDate(page.updated, i18n.locale) || '—'}`}
      </Fact>
    </Box>
  )
}

function Files({ details, fileId }: { details: Details; fileId: number }) {
  const { t, i18n } = useLingui()
  const files = currentFiles(details.files ?? [], fileId)
  const installed = files[0]?.fileId === fileId ? files[0] : undefined
  const others = installed ? files.slice(1) : files
  const line = (f: (typeof files)[number]) =>
    [f.version, formatSize(f.sizeKb, i18n.locale), formatDate(f.uploaded, i18n.locale)]
      .filter(Boolean)
      .join(' · ')
  return (
    <>
      {installed ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
          <Typography sx={heading}>{t`Installed file`}</Typography>
          <Typography sx={{ ...text, overflowWrap: 'anywhere' }}>
            {installed.name || installed.fileName}
          </Typography>
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
              <Typography noWrap={true} sx={{ ...text, flex: 1, minWidth: 0 }}>
                {f.name || f.fileName}
              </Typography>
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
}: {
  details: Details
  mod: Mod
  fileId: number
  modId: number
}) {
  const { t } = useLingui()
  const { page } = details
  const logs = details.changelogs ?? []
  const description = page.description ? parseBBCode(page.description) : []
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      {page.adult || !page.available ? (
        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
          {page.adult ? (
            <Chip size="small" color="warning" variant="outlined" label={t`Adult content`} />
          ) : null}
          {page.available ? null : (
            <Chip
              size="small"
              color="warning"
              variant="outlined"
              icon={<TriangleAlert size={14} aria-hidden={true} />}
              label={t`Not available on Nexus`}
            />
          )}
        </Box>
      ) : null}
      {page.summary ? (
        <Typography sx={{ ...text, color: 'text.secondary' }}>{page.summary}</Typography>
      ) : null}
      <Facts details={details} mod={mod} />
      <NexusAccountActions modId={modId} version={mod.version} endorsement={page.endorsement} />
      <Files details={details} fileId={fileId} />
      {logs.length > 0 ? (
        <Fold title={t`Recent changes`}>
          {recentChangelogs(logs).map((c) => (
            <Box key={c.version}>
              <Typography sx={{ ...text, fontWeight: BOLD }}>{c.version}</Typography>
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
export function NexusDetails({ mod, modId, fileId }: { mod: Mod; modId: number; fileId: number }) {
  const { t } = useLingui()
  const signedIn = useNexus((s) => s.signedIn)
  const entry = useNexusEntry(modId)
  if (entry?.details) {
    return <Loaded details={entry.details} mod={mod} fileId={fileId} modId={modId} />
  }
  if (!entry) {
    return <Typography sx={muted}>{t`Reading the Nexus page…`}</Typography>
  }
  return (
    <Typography sx={muted}>
      {signedIn
        ? t`Could not read the Nexus page: ${entry.error ?? ''}`
        : t`Sign in to Nexus Mods in Settings to see more from this mod's page.`}
    </Typography>
  )
}
