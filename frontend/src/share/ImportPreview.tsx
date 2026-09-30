import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, Typography } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { CircleAlert, Info, TriangleAlert } from 'lucide-react'
import type {
  Mod,
  Problem,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import { LetterTile } from '../mods/parts.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import {
  formatSize,
  isModState,
  MOD_STATES,
  type ModState,
  type ShownPreview,
  type Summary,
} from './logic.ts'

const DIMMED = 0.5

const CHIP: Record<ModState, { bg: string; fg: string }> = {
  installed: { bg: 'rgba(12,223,100,0.2)', fg: '#6ff5a8' },
  download: { bg: 'rgba(43,139,218,0.25)', fg: '#a3d3f7' },
  dependency: { bg: 'rgba(43,139,218,0.25)', fg: '#a3d3f7' },
  later: { bg: 'rgba(200,200,200,0.18)', fg: '#e0e0e0' },
  unavailable: { bg: 'rgba(243,180,22,0.22)', fg: '#F3B416' },
}

const DOT: Record<ModState, string> = {
  installed: '#0CDF64',
  download: '#2B8BDA',
  dependency: '#2B8BDA',
  later: '#c8c8c8',
  unavailable: '#F3B416',
}

function useStateLabel() {
  const { t } = useLingui()
  return (state: ModState): string =>
    ({
      installed: t`Installed`,
      download: t`Download`,
      dependency: t`Dependency`,
      later: t`Check later`,
      unavailable: t`Unavailable`,
    })[state]
}

function Tile({ mod, checked, onToggle }: { mod: Mod; checked: boolean; onToggle: () => void }) {
  const { t } = useLingui()
  const label = useStateLabel()
  const state = isModState(mod.state) ? mod.state : 'later'
  const fixed = state === 'installed' || state === 'unavailable'
  const notes = [
    mod.different ? t`different file` : '',
    mod.unverified ? t`unverified until downloaded` : '',
  ].filter(Boolean)
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        p: 1,
        minWidth: 0,
        bgcolor: 'rgba(55,55,65,0.9)',
        borderRadius: '6px',
        opacity: checked || state === 'unavailable' ? 1 : DIMMED,
      }}
    >
      <Box sx={{ position: 'relative', display: 'flex' }}>
        <LetterTile mod={{ uniqueId: mod.key, name: mod.name }} size={40} />
        <Checkbox
          size="small"
          checked={checked && !fixed}
          disabled={fixed}
          onChange={onToggle}
          slotProps={{ input: { 'aria-label': t`Include ${mod.name}` } }}
          sx={{
            position: 'absolute',
            top: -8,
            left: -8,
            p: 0.25,
            bgcolor: 'rgba(20,20,24,0.8)',
            borderRadius: '4px',
          }}
        />
      </Box>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography noWrap={true} sx={{ fontSize: 14, fontWeight: 600 }} title={mod.name}>
          {mod.name}
        </Typography>
        <Typography noWrap={true} sx={{ fontSize: 12, color: 'text.secondary' }}>
          {[mod.author, ...notes].filter(Boolean).join(' · ')}
        </Typography>
      </Box>
      <Box
        sx={{
          px: 1,
          py: 0.25,
          borderRadius: '4px',
          fontSize: 11,
          fontWeight: 700,
          letterSpacing: '0.04em',
          textTransform: 'uppercase',
          whiteSpace: 'nowrap',
          bgcolor: CHIP[state].bg,
          color: CHIP[state].fg,
        }}
      >
        {label(state)}
      </Box>
    </Box>
  )
}

function openPage(url: string) {
  Browser.OpenURL(url).catch(reportUnexpected)
}

function ProblemRow({
  problem,
  onLeaveOut,
}: {
  problem: Problem
  onLeaveOut: (key: string) => void
}) {
  const { t } = useLingui()
  const { name, detail } = problem
  const texts: Record<string, string> = {
    removed: t`${name} was removed from Nexus`,
    'no-file': t`${name} has no file Mortar can use in place of the one shared`,
    broken: t`${name} is broken for ${detail}`,
    missing: t`${name} needs ${detail}, which Mortar cannot install`,
    free: t`Free account: each Nexus download takes one click on Nexus`,
  }
  const text =
    texts[problem.kind] ??
    t`Some files could not be checked against Nexus, so a few may differ from what was shared.`
  const warn = problem.kind === 'free' || problem.kind === 'unconfirmed'
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1.25,
        px: 1.5,
        py: 0.75,
        bgcolor: 'rgba(0,0,0,0.25)',
        borderRadius: '6px',
        fontSize: 13,
      }}
    >
      {warn ? <Info size={16} color="#2B8BDA" /> : <TriangleAlert size={16} color="#F3B416" />}
      <Box component="span" sx={{ flex: 1, minWidth: 0 }}>
        {text}
      </Box>
      {problem.url ? (
        <Button size="small" variant="text" color="inherit" onClick={() => openPage(problem.url)}>
          {t`Page`}
        </Button>
      ) : null}
      {problem.kind === 'broken' ? (
        <Button
          size="small"
          variant="outlined"
          color="inherit"
          onClick={() => onLeaveOut(problem.key)}
        >
          {t`Leave out`}
        </Button>
      ) : null}
    </Box>
  )
}

export function Tiles({
  mods,
  excluded,
  onToggle,
}: {
  mods: Mod[]
  excluded: ReadonlySet<string>
  onToggle: (key: string) => void
}) {
  return (
    <Box
      sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(210px, 1fr))', gap: 1 }}
    >
      {mods.map((m) => (
        <Tile key={m.key} mod={m} checked={!excluded.has(m.key)} onToggle={() => onToggle(m.key)} />
      ))}
    </Box>
  )
}

export function Problems({
  preview,
  excluded,
  onLeaveOut,
}: {
  preview: ShownPreview
  excluded: ReadonlySet<string>
  onLeaveOut: (key: string) => void
}) {
  const shown = preview.problems.filter((p) => !(p.kind === 'broken' && excluded.has(p.key)))
  if (shown.length === 0) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
      {shown.map((p) => (
        <ProblemRow
          key={`${p.kind}-${p.key}-${p.detail}-${p.name}`}
          problem={p}
          onLeaveOut={onLeaveOut}
        />
      ))}
    </Box>
  )
}

export function StatusBar({ preview, summary }: { preview: ShownPreview; summary: Summary }) {
  const { t } = useLingui()
  const label = useStateLabel()
  const size = formatSize(summary.sizeKb)
  const total = preview.mods.length
  const ready = summary.toImport > 0
  const counts = MOD_STATES.filter((s) => summary.counts[s] > 0)
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 0.75,
        p: 1.5,
        bgcolor: 'rgba(0,0,0,0.3)',
        borderRadius: '8px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1.5, flexWrap: 'wrap' }}>
        <Box
          sx={{ display: 'flex', alignItems: 'center', gap: 0.75, fontSize: 15, fontWeight: 700 }}
        >
          <CircleAlert size={18} />
          {ready ? t`Ready to import` : t`Nothing to download`}
        </Box>
        <Typography sx={{ fontWeight: 600 }}>{preview.name}</Typography>
        <Typography sx={{ color: 'text.secondary', fontSize: 13 }}>
          {ready
            ? t`${plural(total, { one: '# mod', other: '# mods' })} · about ${size} to download`
            : plural(total, { one: '# mod', other: '# mods' })}
        </Typography>
        {preview.settings > 0 ? (
          <Typography sx={{ color: 'text.secondary', fontSize: 13 }}>
            {plural(preview.settings, {
              one: 'with # settings file, written once its mod is installed',
              other: 'with # settings files, written once their mods are installed',
            })}
          </Typography>
        ) : null}
      </Box>
      <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', fontSize: 13 }}>
        {counts.map((s) => (
          <Box
            key={s}
            component="span"
            sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75 }}
          >
            <Box
              component="span"
              sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: DOT[s] }}
            />
            {summary.counts[s]} {label(s).toLowerCase()}
          </Box>
        ))}
        {summary.leftOut > 0 ? <Box component="span">{t`${summary.leftOut} left out`}</Box> : null}
      </Box>
    </Box>
  )
}
