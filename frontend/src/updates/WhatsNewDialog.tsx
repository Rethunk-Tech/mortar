import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Link,
  Typography,
} from '@mui/material'
import {
  Download,
  Gamepad2,
  type LucideIcon,
  RefreshCw,
  Rocket,
  Save,
  Search,
  Settings2,
  Sparkles,
  TriangleAlert,
  Users,
  Wrench,
} from 'lucide-react'
import { useEffect } from 'react'
import type { ReleaseNotes as ReleaseNotesResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/updatesvc/models.ts'
import {
  AckWhatsNew,
  WhatsNew as LoadWhatsNew,
  ReleaseNotes,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/updatesvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { openPage } from '../mods/menu.ts'
import { PageLink } from '../share/CollectionNotes.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { groupNotes, type NoteArea, type NoteKind, type NoteLine } from './releaseNotes.ts'
import { showWhatsNew, useWhatsNew } from './whatsNew.ts'

const BULLET = /^[-*]\s+/

// Areas the release notes name (build/release/notes.sh groups commits by the same ones); any other title gets the
// sparkle.
const AREA_ICONS: Record<string, LucideIcon> = {
  Games: Gamepad2,
  'Mods and browsing': Search,
  'Profiles and sharing': Users,
  'Launching and loaders': Rocket,
  'Problems and diagnostics': TriangleAlert,
  'Downloads and updates': Download,
  Saves: Save,
  'Interface and settings': Settings2,
}
const KIND_ICONS: Record<NoteKind, LucideIcon> = {
  new: Sparkles,
  changed: RefreshCw,
  fixed: Wrench,
}

function LineText({ line }: { line: NoteLine }) {
  return line.parts.map((part, i) => (
    <PageLink
      key={part.at}
      part={i === 0 ? { ...part, text: part.text.replace(BULLET, '') } : part}
    />
  ))
}

function KindChip({ kind }: { kind: NoteKind }) {
  const { t } = useLingui()
  const label = { new: t`New`, changed: t`Changed`, fixed: t`Fixed` }[kind]
  return (
    <Chip
      size="small"
      label={label}
      color={kind === 'new' ? 'primary' : 'default'}
      variant={kind === 'new' ? 'filled' : 'outlined'}
      sx={{ height: 20, fontSize: 11, fontWeight: 600, flexShrink: 0, minWidth: 64, mt: '1px' }}
    />
  )
}

function AreaSection({ area }: { area: NoteArea }) {
  const { t } = useLingui()
  const kindTitles: Record<NoteKind, string> = { new: t`New`, changed: t`Changed`, fixed: t`Fixed` }
  const title = area.byKind ? kindTitles[area.title as NoteKind] : area.title
  const Icon = area.byKind
    ? KIND_ICONS[area.title as NoteKind]
    : (AREA_ICONS[area.title] ?? Sparkles)
  return (
    <Box component="section" sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: 'text.secondary' }}>
        <Icon size={16} aria-hidden={true} />
        <Typography
          component="h3"
          sx={{
            fontSize: 12,
            fontWeight: 700,
            letterSpacing: '0.06em',
            textTransform: 'uppercase',
          }}
        >
          {title}
        </Typography>
        <Typography sx={{ fontSize: 12 }}>{area.items.length}</Typography>
      </Box>
      <Box
        component="ul"
        sx={{ m: 0, p: 0, listStyle: 'none', display: 'flex', flexDirection: 'column', gap: 1 }}
      >
        {area.items.map((item) => (
          <Box
            component="li"
            key={item.line.at}
            sx={{ display: 'flex', gap: 1.25, alignItems: 'flex-start' }}
          >
            {area.byKind ? null : <KindChip kind={item.kind} />}
            <Typography sx={{ fontSize: 14, lineHeight: 1.45, color: 'var(--mortar-ink-sec)' }}>
              <LineText line={item.line} />
            </Typography>
          </Box>
        ))}
      </Box>
    </Box>
  )
}

function Notes({ text }: { text: string }) {
  const { t } = useLingui()
  const { areas, extra } = groupNotes(text)
  const items = areas.flatMap((a) => a.items)
  const count = (kind: NoteKind) => items.filter((i) => i.kind === kind).length
  const summary = [
    count('new')
      ? plural({ count: count('new') }, { one: '# new feature', other: '# new features' })
      : '',
    count('changed')
      ? plural({ count: count('changed') }, { one: '# change', other: '# changes' })
      : '',
    count('fixed') ? plural({ count: count('fixed') }, { one: '# fix', other: '# fixes' }) : '',
  ].filter(Boolean)
  if (areas.length === 0) {
    return (
      <Typography sx={{ whiteSpace: 'pre-wrap', fontSize: 14, color: 'var(--mortar-ink-sec)' }}>
        {text || t`No notes for this release.`}
      </Typography>
    )
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
      {summary.length > 0 ? (
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
          {summary.join(' · ')}
        </Typography>
      ) : null}
      {areas.map((area) => (
        <AreaSection key={`${area.byKind}:${area.title}`} area={area} />
      ))}
      {extra.length > 0 ? (
        <Box
          sx={{
            borderTop: '1px solid var(--mortar-hairline-faint)',
            pt: 1.5,
            display: 'flex',
            flexDirection: 'column',
            gap: 0.5,
          }}
        >
          {extra.map((line) => (
            <Typography key={line.at} sx={{ fontSize: 12, color: 'text.secondary' }}>
              <LineText line={line} />
            </Typography>
          ))}
        </Box>
      ) : null}
    </Box>
  )
}

export function WhatsNewDialog() {
  const { t } = useLingui()
  const shown = useWhatsNew((s) => s)
  const version = shown?.version
  const { data: release } = useLoaded<ReleaseNotesResult | null>(
    version ? () => ReleaseNotes(version) : null,
    [version],
    null,
    reportUnexpected,
  )
  useEffect(() => {
    LoadWhatsNew()
      .then((w) =>
        w.notes && w.version ? showWhatsNew(w.version, w.notes, w.earlier ?? []) : undefined,
      )
      .catch(reportUnexpected)
  }, [])
  const close = () => {
    useWhatsNew.setState(null, true)
    AckWhatsNew().catch(reportUnexpected)
  }
  return (
    <Dialog open={shown !== null} onClose={close} fullWidth={true} maxWidth="md">
      <DialogTitle>{t`What is new in Mortar ${{ version: version ?? '' }}`}</DialogTitle>
      <DialogContent>
        <Notes text={release?.available ? release.notes : (shown?.fallback ?? '')} />
        {release && !release.available ? (
          <Link
            component="button"
            type="button"
            sx={{ mt: 1.5, fontSize: 13 }}
            onClick={() =>
              openPage(`https://github.com/Rethunk-Tech/mortar/releases/tag/v${version}`)
            }
          >
            {t`Read the release notes on ${{ source: sourceLabel('github') }}`}
          </Link>
        ) : null}
        {(shown?.earlier ?? []).map((r) => (
          <Box
            key={r.version}
            component="section"
            sx={{
              mt: space.pad,
              pt: space.pad,
              borderTop: '1px solid var(--mortar-hairline-faint)',
            }}
          >
            <Typography component="h2" sx={{ fontSize: 16, fontWeight: 700, mb: 1.5 }}>
              {t`Also in Mortar ${{ version: r.version }}`}
            </Typography>
            <Notes text={r.notes} />
          </Box>
        ))}
      </DialogContent>
      <DialogActions>
        <Button variant="contained" onClick={close}>
          {t`Close`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
