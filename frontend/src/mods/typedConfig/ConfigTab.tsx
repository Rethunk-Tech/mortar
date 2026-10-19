import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Chip,
  List,
  ListItemButton,
  ListItemText,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material'
import { Check, ChevronDown, Filter, Play, SlidersHorizontal } from 'lucide-react'
import { memo, useCallback, useEffect, useMemo, useState } from 'react'
import type {
  Hit,
  ModConfig,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/models.ts'
import { Search } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/service.ts'
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
import { flattenModGroups, useModVirtual, type VirtualRow } from '../virtualRows.ts'
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
const MIN_SETTING_QUERY = 2
const SETTING_SEARCH_DELAY_MS = 300
const DIMMED = 0.55
const SHOWS: ConfigShow[] = ['all', 'changed', 'menu', 'waiting']

function ChipLabel({ chip }: { chip: NonNullable<ConfigChip> }) {
  const { t } = useLingui()
  if (chip.kind === 'waiting') {
    return <>{t`${chip.n} waiting`}</>
  }
  return <>{chip.kind === 'changed' ? t`Changed` : t`In-game menu`}</>
}

const ModRow = memo(function ConfigModRow({
  config,
  picture,
  selected,
  onChoose,
}: {
  config: ModConfig
  picture: string | undefined
  selected: boolean
  onChoose: (selection: string) => void
}) {
  const chip = chipOf(config)
  return (
    <ListItemButton
      selected={selected}
      onClick={() => onChoose(modSelection(config.id))}
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
})

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
  pictures,
  onChoose,
}: {
  item: Listed
  view: GroupsView
  heading: ReturnType<typeof useListHeading>
  headerGroups: { key: string; items: GroupsView['groups'][number]['items'] }[]
  selection: string | undefined
  pictures: ReadonlyMap<string, string | undefined>
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
      picture={pictures.get(item.item.id)}
      selected={modSelection(item.item.id) === selection}
      onChoose={onChoose}
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

// Shares the mod groups' collapse state under a key no group name can take.
const OTHER_GROUP_KEY = '\u0000loader-and-other'

function OtherFiles({
  view,
  other,
  selection,
  onChoose,
}: {
  view: GroupsView
  other: { name: string; changed?: boolean }[]
  selection: string | undefined
  onChoose: (selection: string) => void
}) {
  const { t } = useLingui()
  return (
    <>
      <GroupHeaderRow
        groupKey={OTHER_GROUP_KEY}
        count={other.length}
        label={t`Loader and other`}
        collapsed={view.collapsed}
        gameId={view.gameId}
        setCollapsed={view.setCollapsed}
        groupBy="none"
        tagHint=""
        groups={[]}
      />
      {view.collapsed[OTHER_GROUP_KEY] === true
        ? null
        : other.map((f) => (
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

// Memoised: typing in the filter re-renders the tab on every key, and the open editor does not depend on the text.
const Editor = memo(function EditorView({
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
  const jump = useConfigList((s) => s.jump)
  const chosenMod = all.find((m) => modSelection(m.id) === selection)
  if (chosenMod) {
    const landing =
      jump && jump.profile === profile.id && jump.id === chosenMod.id ? jump : undefined
    return (
      <ConfigPane
        key={`${profile.id}/${chosenMod.id}/${landing?.n ?? ''}`}
        {...(landing ? { landing: { file: landing.file, query: landing.query } } : {})}
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
})

// Settings across the profile's config files that the Filter mods text matches, found by the app a moment after typing.
function useSettingHits(game: string, profile: string, query: string): Hit[] {
  const [hits, setHits] = useState<Hit[]>([])
  useEffect(() => {
    const q = query.trim()
    if (q.length < MIN_SETTING_QUERY) {
      // The same empty list keeps the tab from rendering again for every key typed below the minimum.
      setHits((prev) => (prev.length === 0 ? prev : []))
      return
    }
    let live = true
    const timer = setTimeout(() => {
      Search(game, profile, q)
        .then((found) => live && setHits(found ?? []))
        .catch(() => live && setHits((prev) => (prev.length === 0 ? prev : [])))
    }, SETTING_SEARCH_DELAY_MS)
    return () => {
      live = false
      clearTimeout(timer)
    }
  }, [game, profile, query])
  return hits
}

// A search can match hundreds of settings; the list shows the first ones and says how many it left out.
const MAX_SHOWN_HITS = 40

function SettingHits({ profileId, hits }: { profileId: string; hits: Hit[] }) {
  const { t } = useLingui()
  const jumpTo = useConfigList((s) => s.jumpTo)
  if (hits.length === 0) {
    return null
  }
  return (
    <Box role="group" aria-label={t`Matching settings`}>
      <Typography sx={{ px: space.pad, py: 0.5, fontSize: 12, color: 'text.secondary' }}>
        {t`Matching settings`}
      </Typography>
      {hits.slice(0, MAX_SHOWN_HITS).map((h) => (
        <ListItemButton
          key={`${h.modId}/${h.file}/${h.section}/${h.key}`}
          onClick={() => jumpTo(profileId, h.modId, h.file, h.key)}
          sx={{ minHeight: space.row }}
        >
          <ListItemText
            primary={h.label || h.key}
            secondary={h.modName}
            slotProps={{ primary: { noWrap: true }, secondary: { noWrap: true } }}
          />
        </ListItemButton>
      ))}
      {hits.length > MAX_SHOWN_HITS ? (
        <Typography sx={{ px: space.pad, py: 0.5, fontSize: 12, color: 'text.secondary' }}>
          {plural(hits.length - MAX_SHOWN_HITS, {
            one: '# more setting; type more to narrow the search.',
            other: '# more settings; type more to narrow the search.',
          })}
        </Typography>
      ) : null}
    </Box>
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
  hits,
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
  hits: Hit[]
}) {
  const { t } = useLingui()
  const choose = useConfigList((s) => s.select)
  const heading = useListHeading(view.groupBy)
  const { listed, headerGroups } = useListed(view, shown)
  const onChoose = useCallback((sel: string) => choose(profileId, sel), [choose, profileId])
  const mods = useMods((s) => s.mods)
  const pictures = useMemo(() => new Map(mods.map((m) => [m.id, m.picture])), [mods])
  // Hundreds of mods can have settings; only the rows in view are mounted.
  const { parentRef, virtualizer } = useModVirtual(listed, 0)
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
      <List
        ref={parentRef}
        component="div"
        role="group"
        aria-label={t`Mods with settings`}
        sx={{ flex: 1, overflowY: 'auto', py: 0 }}
      >
        <Box sx={{ height: virtualizer.getTotalSize(), position: 'relative' }}>
          {virtualizer.getVirtualItems().map((vi) => {
            const item = listed[vi.index]
            return item ? (
              <Box
                key={item.key}
                ref={virtualizer.measureElement}
                data-index={vi.index}
                sx={{ position: 'absolute', top: 0, left: 0, width: '100%' }}
                style={{ transform: `translateY(${vi.start}px)` }}
              >
                <ListEntry
                  item={item}
                  view={view}
                  heading={heading}
                  headerGroups={headerGroups}
                  selection={selection}
                  pictures={pictures}
                  onChoose={onChoose}
                />
              </Box>
            ) : null
          })}
        </Box>
        {shown.length === 0 && all.length > 0 ? (
          <Typography sx={{ px: space.pad, py: space.gap, color: 'text.secondary' }}>
            {t`No mods match`}
          </Typography>
        ) : null}
        <SettingHits profileId={profileId} hits={hits} />
        {other.length > 0 && showOther && !filtered ? (
          <OtherFiles view={view} other={other} selection={selection} onChoose={onChoose} />
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
  const hits = useSettingHits(game, profile.id, query)
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
          hits={hits}
        />
        <Editor profile={profile} game={game} selection={selection} all={all} other={other} />
      </Box>
    </Box>
  )
}
