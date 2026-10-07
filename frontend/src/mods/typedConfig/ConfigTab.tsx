import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Chip,
  List,
  ListItemButton,
  ListItemText,
  ListSubheader,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material'
import { Check, ChevronDown, Filter, Play, SlidersHorizontal } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ModConfig } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/models.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { playOpenProfile } from '../../launch/playOpen.ts'
import { ControlsRow } from '../../shell/ControlsRow.tsx'
import { EmptyState } from '../../shell/EmptyState.tsx'
import { LoadErrorRow, LoadingRow } from '../../shell/LoadingRow.tsx'
import { SearchField } from '../../shell/SearchField.tsx'
import { space } from '../../theme/density.ts'
import { type InlineError, inlineError } from '../../toasts/report.ts'
import { GroupHeaderRow } from '../ModsGroupHeader.tsx'
import { LetterTile } from '../parts.tsx'
import { useMods } from '../store.ts'
import { useListHeading } from '../useListHeading.ts'
import { useModGroups } from '../useModGroups.ts'
import { flattenModGroups, type VirtualRow } from '../virtualRows.ts'
import { ConfigPane } from './ConfigPane.tsx'
import {
  type ConfigChip,
  type ConfigShow,
  chipOf,
  fileSelection,
  filterConfigMods,
  modSelection,
  selectionOf,
  useConfigList,
} from './configList.ts'

const LIST_WIDTH_PX = 300
const DIMMED = 0.55
const SHOWS: ConfigShow[] = ['all', 'changed', 'menu', 'waiting']

function ChipLabel({ chip }: { chip: NonNullable<ConfigChip> }) {
  const { t } = useLingui()
  if (chip.kind === 'waiting') {
    return <>{t`${chip.n} waiting`}</>
  }
  return <>{chip.kind === 'changed' ? t`Changed` : t`In-game menu`}</>
}

function ModRow({
  config,
  selected,
  onSelect,
}: {
  config: ModConfig
  selected: boolean
  onSelect: () => void
}) {
  const picture = useMods((s) => s.mods.find((m) => m.id === config.id)?.picture)
  const chip = chipOf(config)
  return (
    <ListItemButton
      selected={selected}
      onClick={onSelect}
      sx={{ minHeight: space.row, gap: 1, opacity: config.enabled ? 1 : DIMMED }}
    >
      <LetterTile
        mod={{ id: config.id, name: config.name, ...(picture ? { picture } : {}) }}
        size={24}
      />
      <ListItemText
        primary={config.name}
        slotProps={{ primary: { noWrap: true, title: config.name } }}
      />
      {chip ? (
        <Chip
          size="small"
          color={chip.kind === 'waiting' ? 'warning' : 'default'}
          variant="outlined"
          label={<ChipLabel chip={chip} />}
        />
      ) : null}
    </ListItemButton>
  )
}

function ShowMenu({ show, onChange }: { show: ConfigShow; onChange: (show: ConfigShow) => void }) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const labels: Record<ConfigShow, string> = {
    all: t`All`,
    changed: t`Changed`,
    menu: t`In-game menu`,
    waiting: t`Waiting for next launch`,
  }
  return (
    <>
      <Button
        variant="outlined"
        color={show === 'all' ? 'inherit' : 'primary'}
        startIcon={<Filter size={14} />}
        endIcon={<ChevronDown size={14} aria-hidden={true} />}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
      >
        {show === 'all' ? t`Show` : labels[show]}
      </Button>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={() => setAnchor(null)}>
        {SHOWS.map((item) => (
          <MenuItem
            key={item}
            role="menuitemradio"
            aria-checked={show === item}
            selected={show === item}
            onClick={() => {
              onChange(item)
              setAnchor(null)
            }}
          >
            <Box sx={{ width: 24, display: 'flex' }}>
              {show === item ? <Check size={16} aria-hidden={true} /> : null}
            </Box>
            {labels[item]}
          </MenuItem>
        ))}
      </Menu>
    </>
  )
}

type Listed = VirtualRow<ModConfig>
type GroupsView = ReturnType<typeof useModGroups>

function ListEntry({
  item,
  view,
  heading,
  headerGroups,
  selection,
  onChoose,
}: {
  item: Listed
  view: GroupsView
  heading: ReturnType<typeof useListHeading>
  headerGroups: { key: string; items: GroupsView['groups'][number]['items'] }[]
  selection: string | undefined
  onChoose: (selection: string) => void
}) {
  const { t } = useLingui()
  if (item.kind === 'header') {
    return (
      <GroupHeaderRow
        groupKey={item.groupKey}
        count={item.count}
        label={heading(item.groupKey)}
        collapsed={view.collapsed}
        gameId={view.gameId}
        setCollapsed={view.setCollapsed}
        groupBy={view.groupBy}
        tagHint={t`A mod with several tags appears under its first tag.`}
        groups={headerGroups}
      />
    )
  }
  if (item.kind !== 'row') {
    return null
  }
  return (
    <ModRow
      config={item.item}
      selected={modSelection(item.item.id) === selection}
      onSelect={() => onChoose(modSelection(item.item.id))}
    />
  )
}

function WithoutConfig({ count }: { count: number }) {
  return (
    <Typography sx={{ px: space.pad, py: space.gap, fontSize: 12, color: 'text.secondary' }}>
      {plural(count, {
        one: '# mod has no config yet; mods create it the first time the game runs.',
        other: '# mods have no config yet; mods create it the first time the game runs.',
      })}
    </Typography>
  )
}

function OtherFiles({
  other,
  selection,
  onChoose,
}: {
  other: { name: string; changed?: boolean }[]
  selection: string | undefined
  onChoose: (selection: string) => void
}) {
  const { t } = useLingui()
  return (
    <>
      <ListSubheader sx={{ bgcolor: 'transparent', lineHeight: '32px' }}>
        {t`Loader and other`}
      </ListSubheader>
      {other.map((f) => (
        <ListItemButton
          key={f.name}
          selected={fileSelection(f.name) === selection}
          onClick={() => onChoose(fileSelection(f.name))}
          sx={{ minHeight: space.row }}
        >
          <ListItemText primary={f.name} slotProps={{ primary: { noWrap: true } }} />
          {f.changed ? <Chip size="small" variant="outlined" label={t`Changed`} /> : null}
        </ListItemButton>
      ))}
    </>
  )
}

// The list follows the Mods page's grouping and sort: its groups, filtered to the mods with a config source.
function useListed(view: GroupsView, shown: ModConfig[]) {
  const listed = useMemo(() => {
    const byMod = new Map(shown.map((c) => [`${c.key}/${c.id}`, c]))
    const rows = view.groups
      .map((g) => ({
        key: g.key,
        items: g.items.flatMap((r) => {
          const config = byMod.get(`${r.mod.key}/${r.mod.id}`)
          return config ? [config] : []
        }),
      }))
      .filter((g) => g.items.length > 0)
    return flattenModGroups(rows, {
      grouped: view.groupBy !== 'none',
      collapsed: view.collapsed,
      idOf: (c) => c.id,
    })
  }, [shown, view.groups, view.groupBy, view.collapsed])
  const headerGroups = useMemo(
    () => view.groups.map((g) => ({ key: g.key, items: g.items })),
    [view.groups],
  )
  return { listed, headerGroups }
}

function Editor({
  profile,
  game,
  selection,
  all,
  other,
}: {
  profile: Profile
  game: string
  selection: string | undefined
  all: ModConfig[]
  other: { name: string }[]
}) {
  const mods = useMods((s) => s.mods)
  const chosenMod = all.find((m) => modSelection(m.id) === selection)
  if (chosenMod) {
    return (
      <ConfigPane
        key={`${profile.id}/${chosenMod.id}`}
        name={chosenMod.name}
        mod={mods.find((m) => m.key === chosenMod.key && m.id === chosenMod.id) ?? null}
        target={{ game, profile: profile.id, key: chosenMod.key, id: chosenMod.id }}
      />
    )
  }
  const chosenFile = other.find((f) => fileSelection(f.name) === selection)
  if (!chosenFile) {
    return null
  }
  return (
    <ConfigPane
      key={`${profile.id}/file/${chosenFile.name}`}
      name={chosenFile.name}
      mod={null}
      target={{ game, profile: profile.id, key: '', id: '' }}
      file={chosenFile.name}
    />
  )
}

function NoSettings() {
  const { t } = useLingui()
  return (
    <EmptyState
      icon={<SlidersHorizontal size={40} />}
      title={t`No mod settings yet`}
      action={
        <Button variant="contained" startIcon={<Play size={16} />} onClick={playOpenProfile}>
          {t`Play`}
        </Button>
      }
    >
      {t`Mods write their settings the first time the game runs.`}
    </EmptyState>
  )
}

function ModsColumn({
  profileId,
  view,
  shown,
  all,
  other,
  without,
  selection,
  filtered,
  showOther,
}: {
  profileId: string
  view: GroupsView
  shown: ModConfig[]
  all: ModConfig[]
  other: { name: string; changed?: boolean }[]
  without: number
  selection: string | undefined
  filtered: boolean
  showOther: boolean
}) {
  const { t } = useLingui()
  const choose = useConfigList((s) => s.select)
  const heading = useListHeading(view.groupBy)
  const { listed, headerGroups } = useListed(view, shown)
  const onChoose = (sel: string) => choose(profileId, sel)
  return (
    <Box
      sx={{
        width: LIST_WIDTH_PX,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        borderRight: '1px solid var(--mortar-hairline)',
      }}
    >
      <List aria-label={t`Mods with settings`} sx={{ flex: 1, overflowY: 'auto', py: 0 }}>
        {listed.map((item) => (
          <ListEntry
            key={item.key}
            item={item}
            view={view}
            heading={heading}
            headerGroups={headerGroups}
            selection={selection}
            onChoose={onChoose}
          />
        ))}
        {shown.length === 0 && all.length > 0 ? (
          <Typography sx={{ px: space.pad, py: space.gap, color: 'text.secondary' }}>
            {t`No mods match`}
          </Typography>
        ) : null}
        {other.length > 0 && showOther && !filtered ? (
          <OtherFiles other={other} selection={selection} onChoose={onChoose} />
        ) : null}
      </List>
      {without > 0 ? <WithoutConfig count={without} /> : null}
    </Box>
  )
}

// The Config page: every mod with a config source on the left, the editor of the chosen one on the right.
export function ConfigTab({ profile, game }: { profile: Profile; game: string }) {
  const { t } = useLingui()
  const list = useConfigList((s) => s.byProfile[profile.id]?.list)
  const stored = useConfigList((s) => s.selected[profile.id])
  const load = useConfigList((s) => s.load)
  const mods = useMods((s) => s.mods)
  const view = useModGroups(mods, profile)
  const [query, setQuery] = useState('')
  const [show, setShow] = useState<ConfigShow>('all')
  const stamp = String(profile.updated)
  const [failure, setFailure] = useState<InlineError | null>(null)
  const reload = useCallback(() => {
    setFailure(null)
    load(game, profile.id, stamp, true).catch((e: unknown) => setFailure(inlineError(e)))
  }, [load, game, profile.id, stamp])
  useEffect(reload, [reload])
  const all = list?.mods ?? []
  const total = all.length
  const other = list?.other ?? []
  const shown = useMemo(() => filterConfigMods(all, query, show), [all, query, show])
  const selection = selectionOf(list, stored, profile.id)
  if (!list) {
    return failure ? (
      <LoadErrorRow error={failure} onRetry={reload} />
    ) : (
      <LoadingRow>{t`Reading mod settings…`}</LoadingRow>
    )
  }
  if (all.length === 0 && other.length === 0) {
    return <NoSettings />
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <ControlsRow>
        <SearchField
          label={t`Filter mods`}
          placeholder={plural(total, { one: 'Filter # mod', other: 'Filter # mods' })}
          value={query}
          onChange={setQuery}
          grow={true}
        />
        <ShowMenu show={show} onChange={setShow} />
      </ControlsRow>
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex' }}>
        <ModsColumn
          profileId={profile.id}
          view={view}
          shown={shown}
          all={all}
          other={other}
          without={list.without}
          selection={selection}
          filtered={query.trim() !== ''}
          showOther={show === 'all'}
        />
        <Editor profile={profile} game={game} selection={selection} all={all} other={other} />
      </Box>
    </Box>
  )
}
