import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import {
  ChevronLeft,
  ChevronRight,
  Gauge,
  House,
  List,
  type LucideIcon,
  Save,
  Search,
  SlidersHorizontal,
  SquareTerminal,
  Table2,
  TriangleAlert,
} from 'lucide-react'
import { type ReactNode, useEffect } from 'react'
import { PlayControl } from '../launch/PlayControl.tsx'
import { useBadges } from '../mods/badges.ts'
import { problemCount } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { useLoadProblemsOnFocus } from '../mods/useLoadProblemsOnFocus.ts'
import { openProfileOf, useProfileLoader, useProfiles } from '../profiles/store.ts'
import { arrowFocus } from '../shell/arrowFocus.ts'
import { OneTip } from '../shell/OneTip.tsx'
import { useRail, useSidebarCollapsed } from './sidebarCollapsed.ts'
import {
  type SidebarEntry,
  type SidebarGroupId,
  sidebarGroups,
  tabUnavailable,
} from './sidebarTabs.ts'
import { PANEL_ID, type TabId, tabDomId, useTab } from './tab.ts'

const WIDTH_PX = 220
const RAIL_PX = 76
const ITEM_HEIGHT_PX = 38
const RAIL_ITEM_WIDTH_PX = 52
const RAIL_ITEM_HEIGHT_PX = 44
const COLLAPSE_HEIGHT_PX = 34
const COLLAPSE_RAIL_HEIGHT_PX = 36
const BADGE_FONT_PX = 12
const BADGE_FONT_RAIL_PX = 10
const SELECTED_WEIGHT = 600
const NORMAL_WEIGHT = 400

const icons: Record<TabId, LucideIcon> = {
  home: House,
  mods: Table2,
  saves: Save,
  config: SlidersHorizontal,
  browse: Search,
  problems: TriangleAlert,
  'load-order': List,
  performance: Gauge,
  console: SquareTerminal,
}

const tours: Partial<Record<TabId, string>> = {
  browse: 'browse-tab',
  mods: 'mods-tab',
  problems: 'problems-tab',
}

function Badge({ badge, rail }: { badge: NonNullable<SidebarEntry['badge']>; rail: boolean }) {
  const tone = badge.tone === 'warning' ? 'warning' : 'primary'
  return (
    <Box
      component="span"
      sx={{
        ml: rail ? 0 : 'auto',
        position: rail ? 'absolute' : 'static',
        top: 3,
        right: 2,
        px: rail ? '5px' : '8px',
        py: rail ? 0 : '1px',
        borderRadius: rail ? '8px' : '10px',
        fontSize: rail ? BADGE_FONT_RAIL_PX : BADGE_FONT_PX,
        fontWeight: 700,
        lineHeight: 1.5,
        color: `${tone}.contrastText`,
        bgcolor: `${tone}.main`,
      }}
    >
      {badge.n}
    </Box>
  )
}

// A collapsed rail shows only icons, so the name (and any count) has to come from the label.
function railLabel(label: string, badge: SidebarEntry['badge']): string {
  return badge ? `${label}, ${badge.n}` : label
}

function Item({
  tab,
  label,
  badge,
  active,
  rail,
}: {
  tab: TabId
  label: string
  badge: SidebarEntry['badge']
  active: boolean
  rail: boolean
}) {
  const setTab = useTab((s) => s.setTab)
  const Icon = icons[tab]
  const button = (
    <ButtonBase
      role="tab"
      id={tabDomId(tab)}
      aria-controls={PANEL_ID}
      aria-selected={active}
      aria-label={rail ? railLabel(label, badge) : undefined}
      data-tour={tours[tab]}
      onClick={() => setTab(tab)}
      sx={{
        position: 'relative',
        justifyContent: rail ? 'center' : 'flex-start',
        gap: '12px',
        height: rail ? RAIL_ITEM_HEIGHT_PX : ITEM_HEIGHT_PX,
        width: rail ? RAIL_ITEM_WIDTH_PX : 'auto',
        mx: rail ? 0 : '8px',
        px: rail ? 0 : '12px',
        flexShrink: 0,
        borderRadius: '8px',
        fontFamily: 'inherit',
        fontSize: 14,
        fontWeight: active ? SELECTED_WEIGHT : NORMAL_WEIGHT,
        whiteSpace: 'nowrap',
        color: active ? 'var(--mortar-ink)' : 'var(--mortar-ink-sec)',
        bgcolor: active ? 'var(--mortar-hairline-muted)' : 'transparent',
        '&:hover': {
          bgcolor: active ? 'var(--mortar-hairline-muted)' : 'var(--mortar-hairline-faint)',
        },
      }}
    >
      {active ? (
        <Box
          component="span"
          aria-hidden={true}
          sx={{
            position: 'absolute',
            left: 0,
            top: 8,
            bottom: 8,
            width: 3,
            borderRadius: '2px',
            bgcolor: 'primary.main',
          }}
        />
      ) : null}
      <Icon size={18} aria-hidden={true} />
      {rail ? null : <span>{label}</span>}
      {badge ? <Badge badge={badge} rail={rail} /> : null}
    </ButtonBase>
  )
  return rail ? (
    <OneTip title={label} placement="right">
      {button}
    </OneTip>
  ) : (
    button
  )
}

function GroupHeading({ children }: { children: ReactNode }) {
  return (
    <Box
      aria-hidden={true}
      sx={{
        px: '20px',
        pt: '12px',
        pb: '4px',
        fontSize: 11,
        fontWeight: 700,
        letterSpacing: '0.08em',
        textTransform: 'uppercase',
        color: 'var(--mortar-ink-dim)',
      }}
    >
      {children}
    </Box>
  )
}

function useLabels(): Record<TabId, string> & Record<SidebarGroupId, string> {
  const { t } = useLingui()
  return {
    home: t`Home`,
    mods: t`Mods`,
    saves: t`Saves`,
    config: t`Config`,
    browse: t`Browse`,
    problems: t`Problems`,
    'load-order': t`Load order`,
    performance: t`Performance`,
    console: t`Console`,
    library: t`Library`,
    get: t`Get mods`,
    health: t`Health`,
    logs: t`Logs`,
  }
}

// The open profile's sections as tabs; Home first, then the groups the loader's capabilities leave.
function Sections({ rail }: { rail: boolean }) {
  const { t } = useLingui()
  const labels = useLabels()
  const loader = useProfileLoader()
  const current = useTab((s) => s.tab)
  const setTab = useTab((s) => s.setTab)
  const openId = useProfiles((s) => s.openId)
  const updates = useBadges((s) => s.byProfile[openId]?.updates ?? 0)
  const problems = useMods((s) => (s.problems === null ? null : problemCount(s.problems)))
  const caps = {
    order: Boolean(loader?.order),
    console: Boolean(loader?.console),
    startup: Boolean(loader?.startup),
  }
  // A section saved from another game or loader is not there; fall back to Mods.
  const unavailable = loader !== undefined && tabUnavailable(current, caps)
  useEffect(() => {
    if (unavailable) {
      setTab('mods')
    }
  }, [unavailable, setTab])
  return (
    <Box
      role="tablist"
      aria-orientation="vertical"
      aria-label={t`Profile sections`}
      data-profile-tabs={true}
      onKeyDown={(e) => arrowFocus(e, '[role="tab"]')}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '1px',
        alignItems: rail ? 'center' : 'stretch',
        minHeight: 0,
        overflowY: 'auto',
      }}
    >
      <Item tab="home" label={labels.home} badge={null} active={current === 'home'} rail={rail} />
      {sidebarGroups(caps, { updates, problems }).map((group) => (
        <Box
          key={group.id}
          // A tablist owns only tabs, so the visible group headings are layout, not a role.
          role="presentation"
          sx={{
            display: 'flex',
            flexDirection: 'column',
            gap: '1px',
            alignItems: rail ? 'center' : 'stretch',
          }}
        >
          {rail ? null : <GroupHeading>{labels[group.id]}</GroupHeading>}
          {group.entries.map((entry) => (
            <Item
              key={entry.id}
              tab={entry.id}
              label={labels[entry.id]}
              badge={entry.badge}
              active={current === entry.id}
              rail={rail}
            />
          ))}
        </Box>
      ))}
    </Box>
  )
}

function CollapseButton({ rail }: { rail: boolean }) {
  const { t } = useLingui()
  const toggle = useSidebarCollapsed((s) => s.toggle)
  const label = rail ? t`Expand sidebar` : t`Collapse sidebar`
  const Chevron = rail ? ChevronRight : ChevronLeft
  const button = (
    <ButtonBase
      aria-label={label}
      onClick={toggle}
      sx={{
        justifyContent: rail ? 'center' : 'flex-start',
        gap: '10px',
        flexShrink: 0,
        height: rail ? COLLAPSE_RAIL_HEIGHT_PX : COLLAPSE_HEIGHT_PX,
        width: rail ? RAIL_ITEM_WIDTH_PX : 'auto',
        mx: rail ? 0 : '8px',
        mb: '8px',
        px: rail ? 0 : '12px',
        borderRadius: '8px',
        fontFamily: 'inherit',
        fontSize: 13,
        color: 'var(--mortar-ink-dim)',
        '&:hover': { bgcolor: 'var(--mortar-hairline-faint)' },
      }}
    >
      <Chevron size={16} aria-hidden={true} />
      {rail ? null : <span>{t`Collapse`}</span>}
    </ButtonBase>
  )
  return rail ? (
    <OneTip title={label} placement="right">
      {button}
    </OneTip>
  ) : (
    button
  )
}

export function Sidebar({ game }: { game: string }) {
  const { rail, narrow } = useRail()
  const profile = useProfiles(openProfileOf)
  useLoadProblemsOnFocus()
  return (
    <Box
      component="nav"
      sx={{
        width: rail ? RAIL_PX : WIDTH_PX,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        gap: '1px',
        pt: '10px',
        alignItems: rail ? 'center' : 'stretch',
        bgcolor: 'var(--mortar-nav)',
        borderRight: '1px solid var(--mortar-hairline-muted)',
      }}
    >
      {profile ? <Sections rail={rail} /> : null}
      <Box sx={{ flex: 1 }} />
      {narrow ? null : <CollapseButton rail={rail} />}
      <Box sx={{ width: '100%' }}>
        <PlayControl game={game} rail={rail} />
      </Box>
    </Box>
  )
}
