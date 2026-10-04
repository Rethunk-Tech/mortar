import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, Typography } from '@mui/material'
import { alpha, type Theme, useTheme } from '@mui/material/styles'

const SUCCESS_CHIP = 0.2
const INFO_CHIP = 0.25
const WARN_CHIP = 0.22
const INFO_ROW = 0.12
const INFO_LINE = 0.4
const WARN_ROW = 0.12
const WARN_LINE = 0.45

import { Download, Info, TriangleAlert } from 'lucide-react'
import type { ReactNode } from 'react'
import type {
  Mod,
  Problem,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import { openPage } from '../mods/menu.ts'
import { LetterTile } from '../mods/parts.tsx'
import {
  formatSize,
  isModState,
  MOD_STATES,
  type ModState,
  type ShownPreview,
  type Summary,
} from './logic.ts'

const DIMMED = 0.5

const CHIP = (th: Theme): Record<ModState, { bg: string; fg: string }> => ({
  installed: { bg: alpha(th.palette.success.main, SUCCESS_CHIP), fg: '#6ff5a8' },
  download: { bg: alpha(th.palette.info.main, INFO_CHIP), fg: '#a3d3f7' },
  dependency: { bg: alpha(th.palette.info.main, INFO_CHIP), fg: '#a3d3f7' },
  later: { bg: 'rgba(200,200,200,0.18)', fg: '#e0e0e0' },
  unavailable: { bg: alpha(th.palette.warning.main, WARN_CHIP), fg: th.palette.warning.main },
})

const DOT = (th: Theme): Record<ModState, string> => ({
  installed: th.palette.success.main,
  download: th.palette.info.main,
  dependency: th.palette.info.main,
  later: '#c8c8c8',
  unavailable: th.palette.warning.main,
})

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
  const chip = CHIP(useTheme())
  const label = useStateLabel()
  const state = isModState(mod.state) ? mod.state : 'later'
  const fixed = state === 'installed' || state === 'unavailable'
  const notes = [
    mod.different ? t`different file` : '',
    mod.unverified ? t`unverified until downloaded` : '',
    mod.site === 'local' && !mod.enabled ? t`switched off` : '',
  ].filter(Boolean)
  const byline = [mod.author, ...notes].filter(Boolean).join(' · ')
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        height: 52,
        pr: 1,
        minWidth: 0,
        bgcolor: 'var(--mortar-card-hover)',
        borderRadius: '3px',
        overflow: 'hidden',
        opacity: checked || state === 'unavailable' ? 1 : DIMMED,
      }}
    >
      <Box sx={{ position: 'relative', display: 'flex', flexShrink: 0 }}>
        <LetterTile mod={{ uniqueId: mod.key, name: mod.name }} size={52} />
        <Checkbox
          size="small"
          checked={checked && !fixed}
          disabled={fixed}
          onChange={onToggle}
          slotProps={{ input: { 'aria-label': t`Include ${mod.name}` } }}
          sx={{
            position: 'absolute',
            top: 3,
            left: 3,
            p: 0,
            bgcolor: 'rgba(20,20,24,0.75)',
            borderRadius: '3px',
            '& .MuiSvgIcon-root': { fontSize: 16 },
          }}
        />
      </Box>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography noWrap={true} sx={{ fontSize: 13, lineHeight: 1.4 }} title={mod.name}>
          {mod.name}
        </Typography>
        <Typography
          noWrap={true}
          sx={{ fontSize: 12, lineHeight: 1.4, color: 'text.secondary' }}
          title={byline}
        >
          {byline}
        </Typography>
      </Box>
      <Box
        sx={{
          flexShrink: 0,
          px: 0.75,
          py: '1px',
          borderRadius: '3px',
          fontSize: 11,
          fontWeight: 700,
          whiteSpace: 'nowrap',
          bgcolor: chip[state].bg,
          color: chip[state].fg,
        }}
      >
        {label(state)}
      </Box>
    </Box>
  )
}

function ProblemRow({
  problem,
  onLeaveOut,
}: {
  problem: Problem
  onLeaveOut: (key: string) => void
}) {
  const { t } = useLingui()
  const theme = useTheme()
  const { name, detail } = problem
  const texts: Record<string, string> = {
    removed: t`${name} was removed from Nexus`,
    'no-file': t`${name} has no file Mortar can use in place of the one shared`,
    broken: t`${name} is broken for ${detail}`,
    missing: t`${name} needs ${detail}, which Mortar cannot install`,
    free: t`Free account: one click on Nexus per download`,
  }
  const text =
    texts[problem.kind] ??
    t`Some files could not be checked against Nexus, so a few may differ from what was shared.`
  const info = problem.kind === 'free' || problem.kind === 'unconfirmed'
  const tone = info
    ? {
        bg: alpha(theme.palette.info.main, INFO_ROW),
        line: alpha(theme.palette.info.main, INFO_LINE),
      }
    : {
        bg: alpha(theme.palette.warning.main, WARN_ROW),
        line: alpha(theme.palette.warning.main, WARN_LINE),
      }
  let icon = <TriangleAlert size={16} color={theme.palette.warning.main} />
  if (problem.kind === 'free') {
    icon = <Download size={16} color="#a3d3f7" />
  } else if (info) {
    icon = <Info size={16} color="#a3d3f7" />
  }
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1.25,
        height: 44,
        px: 1.5,
        minWidth: 0,
        bgcolor: tone.bg,
        border: `1px solid ${tone.line}`,
        borderRadius: '4px',
        fontSize: 13,
        whiteSpace: 'nowrap',
      }}
    >
      {icon}
      <Box
        component="span"
        title={text}
        sx={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}
      >
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

function Pill({
  bg,
  fg,
  dot,
  children,
}: {
  bg: string
  fg: string
  dot: string
  children: ReactNode
}) {
  return (
    <Box
      component="span"
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 0.75,
        px: 1.25,
        py: 0.5,
        borderRadius: '12px',
        fontSize: 13,
        whiteSpace: 'nowrap',
        bgcolor: bg,
        color: fg,
      }}
    >
      <Box component="span" sx={{ width: 8, height: 8, borderRadius: '4px', bgcolor: dot }} />
      {children}
    </Box>
  )
}

function CountLabel({ state, count }: { state: ModState; count: number }) {
  const { t } = useLingui()
  return {
    installed: t`${count} installed`,
    download: t`${count} to download`,
    dependency: plural(count, { one: '# dependency', other: '# dependencies' }),
    later: t`${count} checked later`,
    unavailable: t`${count} unavailable`,
  }[state]
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
      sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))',
        gap: '6px',
      }}
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
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
        gap: '6px',
        p: '8px 8px 0',
        maxHeight: 142,
        overflowY: 'auto',
      }}
    >
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
  const th = useTheme()
  const chip = CHIP(th)
  const dot = DOT(th)
  const total = preview.mods.length
  const ready = summary.toImport > 0
  const mods = plural(total, { one: '# mod', other: '# mods' })
  const size = formatSize(summary.sizeKb)
  const settings = plural(preview.settings, {
    one: 'with # settings file, written once its mod is installed',
    other: 'with # settings files, written once their mods are installed',
  })
  const skippedSettings = preview.skippedSettings ?? []
  const counts = MOD_STATES.filter((s) => summary.counts[s] > 0)
  const detail = [
    ready ? t`${mods} · about ${size} to download` : mods,
    preview.settings > 0 ? settings : '',
  ]
    .filter(Boolean)
    .join(' · ')
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '18px',
        m: 1,
        minHeight: 52,
        px: '18px',
        py: 0.5,
        bgcolor: '#0e1116',
        borderRadius: '4px',
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1.25,
          fontSize: 16,
          color: '#a3d3f7',
          whiteSpace: 'nowrap',
        }}
      >
        <Info size={18} />
        {ready ? t`Ready to import` : t`Nothing to download`}
      </Box>
      <Box sx={{ width: '1px', height: 24, bgcolor: 'var(--mortar-hairline-15)', flexShrink: 0 }} />
      <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <Typography noWrap={true} sx={{ fontSize: 16, fontWeight: 600 }} title={preview.name}>
          {preview.name}
        </Typography>
        <Typography noWrap={true} sx={{ fontSize: 12, color: 'text.secondary' }} title={detail}>
          {detail}
        </Typography>
        {skippedSettings.length > 0 ? (
          <Box>
            <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
              {plural(skippedSettings.length, {
                one: '# settings file was left out (too large):',
                other: '# settings files were left out (too large):',
              })}
            </Typography>
            <Box component="ul" sx={{ m: 0, pl: 2.5, fontSize: 13, color: 'text.secondary' }}>
              {skippedSettings.map((p) => (
                <li key={p}>{p}</li>
              ))}
            </Box>
          </Box>
        ) : null}
      </Box>
      <Box sx={{ flex: 1 }} />
      <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', justifyContent: 'flex-end' }}>
        {counts.map((s) => (
          <Pill key={s} bg={chip[s].bg} fg={chip[s].fg} dot={dot[s]}>
            <CountLabel state={s} count={summary.counts[s]} />
          </Pill>
        ))}
        {summary.leftOut > 0 ? (
          <Pill bg="rgba(200,200,200,0.14)" fg="#e0e0e0" dot="#bdbdbd">
            {t`${summary.leftOut} left out`}
          </Pill>
        ) : null}
      </Box>
    </Box>
  )
}
