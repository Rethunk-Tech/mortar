import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, Typography } from '@mui/material'
import { alpha, type Theme, useTheme } from '@mui/material/styles'
import { formatKb } from '../i18n/bytes.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'

const SUCCESS_CHIP = 0.2
const INFO_CHIP = 0.25
const WARN_CHIP = 0.22

import { Download, Info, TriangleAlert } from 'lucide-react'
import type { ReactNode } from 'react'
import type {
  Mod,
  Problem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import { modsLabel } from '../i18n/counts.ts'
import { openPage } from '../mods/menu.ts'
import { LetterTile } from '../mods/parts.tsx'
import { isModState, MOD_STATES, type ModState, type ShownPreview, type Summary } from './logic.ts'

const DIMMED = 0.5
const LATER_CHIP = 0.18

const CHIP = (th: Theme): Record<ModState, { bg: string; fg: string }> => ({
  installed: { bg: alpha(th.palette.success.main, SUCCESS_CHIP), fg: th.palette.success.light },
  download: { bg: alpha(th.palette.info.main, INFO_CHIP), fg: th.palette.info.light },
  dependency: { bg: alpha(th.palette.info.main, INFO_CHIP), fg: th.palette.info.light },
  later: { bg: alpha(th.palette.text.secondary, LATER_CHIP), fg: th.palette.text.primary },
  unavailable: { bg: alpha(th.palette.warning.main, WARN_CHIP), fg: th.palette.warning.main },
})

// Share-card preview chrome: matches the public share card, not the app theme, so its bar stays dark in the light
// theme too and everything on it is drawn for a dark surface.
const GREY_PILL = { bg: 'rgba(200,200,200,0.14)', fg: '#e0e0e0', dot: '#bdbdbd' }
const INFO_PILL = { bg: 'rgba(43,139,218,0.25)', fg: '#a3d3f7', dot: '#2b8bda' }
const SHARE_CARD = {
  bar: '#0e1116',
  accent: '#a3d3f7',
  text: '#ffffff',
  muted: 'rgba(255,255,255,0.72)',
  divider: 'rgba(255,255,255,0.15)',
  leftOut: GREY_PILL,
  chip: {
    installed: { bg: 'rgba(12,223,100,0.2)', fg: '#7ef0ad', dot: '#0cdf64' },
    download: INFO_PILL,
    dependency: INFO_PILL,
    later: GREY_PILL,
    unavailable: { bg: 'rgba(243,180,22,0.22)', fg: '#f3b416', dot: '#f3b416' },
  } satisfies Record<ModState, { bg: string; fg: string; dot: string }>,
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
  const chip = CHIP(useTheme())
  const label = useStateLabel()
  const state = isModState(mod.state) ? mod.state : 'later'
  const fixed = state === 'installed' || state === 'unavailable'
  const notes = [
    mod.different ? t`different file` : '',
    mod.unverified ? t`unverified until downloaded` : '',
    mod.site === 'local' && !mod.enabled ? t`Disabled`.toLowerCase() : '',
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
        <LetterTile mod={{ id: mod.key, name: mod.name }} size={52} />
        <Checkbox
          size="small"
          checked={checked && !fixed}
          disabled={fixed}
          onChange={onToggle}
          slotProps={{ input: { 'aria-label': t`Include ${{ name: mod.name }}` } }}
          sx={{
            position: 'absolute',
            top: 3,
            left: 3,
            p: 0,
            bgcolor: 'var(--mortar-overlay-80)',
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
  const tone = info ? 'info' : 'warning'
  let icon = <TriangleAlert size={16} color={theme.palette.warning.main} />
  if (problem.kind === 'free') {
    icon = <Download size={16} color={theme.palette.info.light} />
  } else if (info) {
    icon = <Info size={16} color={theme.palette.info.light} />
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
        bgcolor: calloutFill(tone),
        border: '1px solid',
        borderColor: calloutLine(tone),
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
    installed: plural(count, { one: '# mod installed', other: '# mods installed' }),
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

export function StatusBar({
  preview,
  summary,
  needsSignIn,
}: {
  preview: ShownPreview
  summary: Summary
  needsSignIn: boolean
}) {
  const { t } = useLingui()
  const total = preview.mods.length
  const ready = summary.toImport > 0
  let status = ready ? t`Ready to import` : t`Nothing to download`
  if (needsSignIn) {
    status = t`Sign in to import`
  }
  const size = formatKb(summary.sizeKb)
  const settings = plural(preview.settings, {
    one: 'with # settings file, written once its mod is installed',
    other: 'with # settings files, written once their mods are installed',
  })
  const skippedSettings = preview.skippedSettings ?? []
  const counts = MOD_STATES.filter((s) => summary.counts[s] > 0)
  const detail = [
    // A mod whose source reports no size adds 0, so a 0 total means unknown rather than nothing to fetch.
    ready && summary.sizeKb > 0
      ? plural(total, {
          one: `# mod · about ${size} to download`,
          other: `# mods · about ${size} to download`,
        })
      : modsLabel(total),
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
        bgcolor: SHARE_CARD.bar,
        color: SHARE_CARD.text,
        borderRadius: '4px',
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1.25,
          fontSize: 16,
          color: SHARE_CARD.accent,
          whiteSpace: 'nowrap',
        }}
      >
        <Info size={18} />
        {status}
      </Box>
      <Box sx={{ width: '1px', height: 24, bgcolor: SHARE_CARD.divider, flexShrink: 0 }} />
      <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <Typography noWrap={true} sx={{ fontSize: 16, fontWeight: 600 }} title={preview.name}>
          {preview.name}
        </Typography>
        <Typography noWrap={true} sx={{ fontSize: 12, color: SHARE_CARD.muted }} title={detail}>
          {detail}
        </Typography>
        {skippedSettings.length > 0 ? (
          <Box>
            <Typography variant="body2" sx={{ fontWeight: 600, color: SHARE_CARD.muted }}>
              {plural(skippedSettings.length, {
                one: '# settings file was left out (too large):',
                other: '# settings files were left out (too large):',
              })}
            </Typography>
            <Box component="ul" sx={{ m: 0, pl: 2.5, fontSize: 13, color: SHARE_CARD.muted }}>
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
          <Pill key={s} {...SHARE_CARD.chip[s]}>
            <CountLabel state={s} count={summary.counts[s]} />
          </Pill>
        ))}
        {summary.leftOut > 0 ? (
          <Pill {...SHARE_CARD.leftOut}>{t`${summary.leftOut} left out`}</Pill>
        ) : null}
      </Box>
    </Box>
  )
}
