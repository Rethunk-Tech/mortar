import { plural } from '@lingui/core/macro'

import { useLingui } from '@lingui/react/macro'

import { Box, Button, Typography } from '@mui/material'

import { useTheme } from '@mui/material/styles'
import { Download, Info, TriangleAlert } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Problem } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { formatKb } from '../i18n/bytes.ts'
import { modsLabel } from '../i18n/counts.ts'
import { listNames } from '../i18n/list.ts'
import { openPage } from '../mods/menu.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'

import { space } from '../theme/density.ts'

import { carriedBy, MOD_STATES, type ModState, type ShownPreview, type Summary } from './logic.ts'

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

const cardList = {
  m: 0,
  pl: space.pad,
  fontSize: 'base.fontSize',
  color: SHARE_CARD.muted,
} as const
const cardTitle = { fontSize: 'large.fontSize', fontWeight: 600 } as const
const cardDetail = { fontSize: 'small.fontSize', color: SHARE_CARD.muted } as const

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
    removed: t`${name} was removed from ${{ source: sourceLabel(problem.site) }}`,
    'not-on-source': t`${name} is not on ${{ source: sourceLabel(problem.site) }}`,
    'no-file': t`${name} has no file Mortar can use in place of the one shared`,
    broken: t`${name} is broken for ${detail}`,
    missing: t`${name} needs ${detail}, which Mortar cannot install`,
    free: t`Free account: one click on ${{ source: sourceLabel('nexus') }} per download`,
  }
  const text =
    texts[problem.kind] ??
    t`Some files could not be checked against ${{ source: sourceLabel('nexus') }}, so a few may differ from what was shared.`
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
        gap: space.gap,
        height: 44,
        px: space.pad,
        minWidth: 0,
        bgcolor: calloutFill(tone),
        border: '1px solid',
        borderColor: calloutLine(tone),
        borderRadius: '4px',
        fontSize: 'base.fontSize',
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
        px: space.gap,
        py: 0.5,
        borderRadius: '12px',
        fontSize: 'base.fontSize',
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

// Files an import left out, and why.
function LeftOut({ title, paths }: { title: string; paths: string[] }) {
  return (
    <Box>
      <Typography variant="body2" sx={{ fontWeight: 600, color: SHARE_CARD.muted }}>
        {title}
      </Typography>
      <Box component="ul" sx={cardList}>
        {paths.map((p) => (
          <li key={p}>{p}</li>
        ))}
      </Box>
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
  const { t, i18n } = useLingui()
  const total = preview.mods.length
  const ready = summary.toImport > 0
  let status = ready ? t`Ready to import` : t`Nothing to download`
  if (needsSignIn) {
    status = t`Sign in to import`
  }
  const size = formatKb(summary.sizeKb)
  const settings = plural(
    { count: preview.settings },
    {
      one: 'with # settings file, written once its mod is installed',
      other: 'with # settings files, written once their mods are installed',
    },
  )
  const skippedSettings = preview.skippedSettings ?? []
  const skippedPrograms = preview.skippedPrograms ?? []
  const carried = carriedBy(preview)
  const carriedLabels = {
    notes: t`Notes`.toLocaleLowerCase(i18n.locale),
    configFiles: t`Config files`.toLocaleLowerCase(i18n.locale),
    problemChoices: t`Problem choices`.toLocaleLowerCase(i18n.locale),
  }
  const counts = MOD_STATES.filter((s) => summary.counts[s] > 0)
  const detail = [
    // A mod whose source reports no size adds 0, so a 0 total means unknown rather than nothing to fetch.
    modsLabel(total),
    ready && summary.sizeKb > 0 ? t`about ${size} to download` : '',
    preview.settings > 0 ? settings : '',
    carried.length > 0
      ? t`Includes: ${{ names: listNames(carried.map((c) => carriedLabels[c])) }}`
      : '',
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
          gap: space.gap,
          fontSize: 'large.fontSize',
          color: SHARE_CARD.accent,
          whiteSpace: 'nowrap',
        }}
      >
        <Info size={18} />
        {status}
      </Box>
      <Box sx={{ width: '1px', height: 24, bgcolor: SHARE_CARD.divider, flexShrink: 0 }} />
      <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <Typography noWrap={true} sx={cardTitle} title={preview.name}>
          {preview.name}
        </Typography>
        <Typography noWrap={true} sx={cardDetail} title={detail}>
          {detail}
        </Typography>
        {skippedSettings.length > 0 ? (
          <LeftOut
            title={plural(
              { count: skippedSettings.length },
              {
                one: '# settings file was left out (too large):',
                other: '# settings files were left out (too large):',
              },
            )}
            paths={skippedSettings}
          />
        ) : null}
        {skippedPrograms.length > 0 ? (
          <LeftOut
            title={t`${plural({ count: skippedPrograms.length }, { one: '# program file was', other: '# program files were' })} left out; Mortar copies settings, not programs:`}
            paths={skippedPrograms}
          />
        ) : null}
      </Box>
      <Box sx={{ flex: 1 }} />
      <Box sx={{ display: 'flex', gap: space.gap, flexWrap: 'wrap', justifyContent: 'flex-end' }}>
        {counts.map((s) => (
          <Pill key={s} {...SHARE_CARD.chip[s]}>
            <CountLabel state={s} count={summary.counts[s]} />
          </Pill>
        ))}
        {summary.leftOut > 0 ? (
          <Pill {...SHARE_CARD.leftOut}>{t`${{ leftOut: summary.leftOut }} left out`}</Pill>
        ) : null}
      </Box>
    </Box>
  )
}
