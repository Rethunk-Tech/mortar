import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Checkbox, Link, Typography } from '@mui/material'
import { alpha, type Theme, useTheme } from '@mui/material/styles'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { listNames } from '../i18n/list.ts'
import { lowerFirst } from '../i18n/lowerFirst.ts'
import { formatAuthors } from '../mods/authorNormalize.ts'
import { openPage } from '../mods/menu.ts'
import { LetterTile } from '../mods/parts.tsx'
import { visuallyHidden } from '../shell/visuallyHidden.ts'
import { space } from '../theme/density.ts'
import {
  isModState,
  type ModState,
  needsCurseForgeKey,
  needsItchFile,
  needsPatreonFile,
} from './logic.ts'

const SUCCESS_CHIP = 0.2
const INFO_CHIP = 0.25
const WARN_CHIP = 0.22

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
const tileName = {
  fontSize: 'base.fontSize',
  lineHeight: 1.4,
  overflowWrap: 'anywhere',
  userSelect: 'text',
} as const
const tileDetail = {
  fontSize: 'small.fontSize',
  lineHeight: 1.4,
  color: 'text.secondary',
  userSelect: 'text',
} as const

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
    mod.site === 'local' && !mod.enabled ? lowerFirst(t`Disabled`) : '',
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
        <Typography sx={tileName}>
          <TileName mod={mod} />
        </Typography>
        <Typography sx={tileDetail}>{byline}</Typography>
      </Box>
      <Box
        sx={{
          flexShrink: 0,
          px: 0.75,
          py: '1px',
          borderRadius: '3px',
          fontSize: 'tiny.fontSize',
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
