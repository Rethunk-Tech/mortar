import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Checkbox, Link, Typography } from '@mui/material'
import { alpha, type Theme, useTheme } from '@mui/material/styles'
import { formatKb } from '../i18n/bytes.ts'
import { listNames } from '../i18n/list.ts'
import { formatAuthors } from '../mods/authorNormalize.ts'
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
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { modsLabel } from '../i18n/counts.ts'
import { openPage } from '../mods/menu.ts'
import { LetterTile } from '../mods/parts.tsx'
import { visuallyHidden } from '../shell/visuallyHidden.ts'
import { space } from '../theme/density.ts'
import {
  carriedBy,
  isModState,
  MOD_STATES,
  type ModState,
  needsCurseForgeKey,
  needsItchFile,
  needsPatreonFile,
  type ShownPreview,
  type Summary,
} from './logic.ts'

const DIMMED = 0.5
const LATER_CHIP = 0.18
const TILE_FILL = 0.28
const TILE_LINE = 0.7
// Installed mods are most of a familiar profile, so their green stays quiet beside the tint of a mod coming in.
const INSTALLED_FILL = 0.1
const INSTALLED_LINE = 0.3

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

// The whole tile is the checkbox, so nothing covers the icon: an installed mod is tinted green, one coming in the
// primary colour, and one left out is dimmed.
function tileTint(
  theme: Theme,
  state: ModState,
  selected: boolean,
): { fill: string; line: string } | null {
  if (state === 'installed') {
    const green = theme.palette.success.main
    return { fill: alpha(green, INSTALLED_FILL), line: alpha(green, INSTALLED_LINE) }
  }
  const primary = theme.palette.primary.main
  return selected ? { fill: alpha(primary, TILE_FILL), line: alpha(primary, TILE_LINE) } : null
}

// The name in full (it wraps rather than cuts off, so it can be copied), opening the mod's page when it has one.
function TileName({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  if (!mod.pageUrl) {
    return mod.name
  }
  return (
    <Link
      component="button"
      underline="hover"
      color="inherit"
      title={t`Open its page in the browser`}
      onClick={(e) => {
        e.stopPropagation()
        openPage(mod.pageUrl)
      }}
      sx={{ font: 'inherit', textAlign: 'left', verticalAlign: 'baseline', userSelect: 'text' }}
    >
      {mod.name}
    </Link>
  )
}

function Tile({ mod, checked, onToggle }: { mod: Mod; checked: boolean; onToggle: () => void }) {
  const { t } = useLingui()
  const theme = useTheme()
  const chip = CHIP(theme)
  const label = useStateLabel()
  const state = isModState(mod.state) ? mod.state : 'later'
  const fixed = state === 'installed' || state === 'unavailable'
  const notes = [
    mod.different ? t`different file` : '',
    mod.unverified ? t`unverified until downloaded` : '',
    mod.site === 'local' && !mod.enabled ? t`Disabled`.toLowerCase() : '',
    needsPatreonFile(mod)
      ? t`Needs a file from its ${{ source: sourceLabel('patreon') }} post`
      : '',
    needsItchFile(mod) ? t`Needs a file from its ${{ source: sourceLabel('itch') }} page` : '',
    needsCurseForgeKey(mod)
      ? t`Needs a ${{ source: sourceLabel('curseforge') }} key, or the file from its page`
      : '',
    mod.filesFrom === 'code' ? t`program files from this code` : '',
    mod.filesFrom === 'computer' ? t`copied from this computer` : '',
  ].filter(Boolean)
  const byline = [formatAuthors(mod.author), ...notes].filter(Boolean).join(' · ')
  const tint = tileTint(theme, state, checked && !fixed)
  return (
    <Box
      onClick={
        fixed
          ? undefined
          : () => {
              // A click that ends a text selection (a name being copied) is not a toggle.
              if (!globalThis.getSelection()?.toString()) {
                onToggle()
              }
            }
      }
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        minHeight: 52,
        pr: space.gap,
        minWidth: 0,
        bgcolor: tint?.fill ?? 'var(--mortar-card-hover)',
        border: '1px solid',
        borderColor: tint?.line ?? 'transparent',
        borderRadius: '3px',
        overflow: 'hidden',
        cursor: fixed ? 'default' : 'pointer',
        opacity: checked || state === 'unavailable' ? 1 : DIMMED,
        '&:focus-within': { outline: `2px solid ${theme.palette.primary.main}`, outlineOffset: 1 },
      }}
    >
      <Checkbox
        checked={checked && !fixed}
        disabled={fixed}
        onChange={onToggle}
        onClick={(e) => e.stopPropagation()}
        slotProps={{ input: { 'aria-label': t`Include ${{ name: mod.name }}` } }}
        sx={visuallyHidden}
      />
      <Box sx={{ display: 'flex', flexShrink: 0, alignSelf: 'flex-start' }}>
        <LetterTile mod={{ id: mod.key, name: mod.name, picture: mod.icon ?? '' }} size={52} />
      </Box>
      <Box sx={{ flex: 1, minWidth: 0, py: 0.5 }}>
        <Typography
          sx={{ fontSize: 13, lineHeight: 1.4, overflowWrap: 'anywhere', userSelect: 'text' }}
        >
          <TileName mod={mod} />
        </Typography>
        <Typography
          sx={{ fontSize: 12, lineHeight: 1.4, color: 'text.secondary', userSelect: 'text' }}
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
      {mod.site === 'patreon' && mod.pageUrl ? (
        <Button
          size="small"
          color="inherit"
          onClick={(e) => {
            e.stopPropagation()
            openPage(mod.pageUrl)
          }}
        >
          {t`Open post`}
        </Button>
      ) : null}
      {(needsItchFile(mod) || needsCurseForgeKey(mod)) && mod.pageUrl ? (
        <Button
          size="small"
          color="inherit"
          onClick={(e) => {
            e.stopPropagation()
            openPage(mod.pageUrl)
          }}
        >
          {t`Open page`}
        </Button>
      ) : null}
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
    removed: t`${name} was removed from ${{ source: sourceLabel('nexus') }}`,
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
        px: space.gap,
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

// Files an import left out, and why.
function LeftOut({ title, paths }: { title: string; paths: string[] }) {
  return (
    <Box>
      <Typography variant="body2" sx={{ fontWeight: 600, color: SHARE_CARD.muted }}>
        {title}
      </Typography>
      <Box component="ul" sx={{ m: 0, pl: space.pad, fontSize: 13, color: SHARE_CARD.muted }}>
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

export function Tiles({
  mods,
  excluded,
  onToggle,
}: {
  mods: Mod[]
  excluded: ReadonlySet<string>
  onToggle: (key: string) => void
}) {
  const { t } = useLingui()
  // A code's own files are programs nobody reviewed: no mod site scanned or signed them, so the player decides first.
  const carried = mods
    .filter((m) => m.filesFrom === 'code' && !excluded.has(m.key))
    .map((m) => m.name)
  return (
    <>
      {carried.length > 0 ? (
        <Alert severity="warning" sx={{ mb: 1 }}>
          {t`${{ names: listNames(carried) }} bring program files with this code rather than from a mod site. Leave them out unless you trust who sent it.`}
        </Alert>
      ) : null}
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))',
          gap: '6px',
        }}
      >
        {mods.map((m) => (
          <Tile
            key={m.key}
            mod={m}
            checked={!excluded.has(m.key)}
            onToggle={() => onToggle(m.key)}
          />
        ))}
      </Box>
    </>
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
