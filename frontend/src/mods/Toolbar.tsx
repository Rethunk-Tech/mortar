import type { I18n } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Tooltip,
  useMediaQuery,
} from '@mui/material'
import {
  Ban,
  Check,
  Download,
  ExternalLink,
  Filter,
  FolderTree,
  Layers,
  Library,
  Plus,
  Settings2,
  Tag,
  ToggleRight,
  User,
} from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { SetListGroupBy } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { compact, compactQuery, searchFieldOpen } from '../game/compact.ts'
import { openDownloadsDialog } from '../install/downloadsDialog.ts'
import { useInstall } from '../install/store.ts'
import { openSettings } from '../nav/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { ViewToggle } from '../shell/ViewToggle.tsx'
import { TipBanner } from '../tips/TipBanner.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { CategoryEditorDialog } from './CategoryEditor.tsx'
import { ExtraFolderMenu } from './ExtraFolderMenu.tsx'
import { onFilterFocus } from './filterFocus.ts'
import { type GroupBy, sanitizeListGroupBy } from './group.ts'
import { openPage } from './menu.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const NEXUS = 'https://www.nexusmods.com/stardewvalley/mods'
const FILTERS: readonly {
  id: Exclude<ModFilter, 'all'>
  label: (i18n: I18n) => string
}[] = [
  { id: 'disabled', label: (i18n) => i18n._(msg`Off`) },
  { id: 'update', label: (i18n) => i18n._(msg`Update available`) },
  { id: 'problem', label: (i18n) => i18n._(msg`Has problems`) },
  { id: 'pinned', label: (i18n) => i18n._(msg`Pinned`) },
  { id: 'local', label: (i18n) => i18n._(msg`Local`) },
  { id: 'recent', label: (i18n) => i18n._(msg`Added this week`) },
]

// At the minimum size the toolbar buttons fold into icon buttons.
const iconWhenCompact = {
  [compact]: {
    minWidth: 36,
    px: 0,
    '& .MuiButton-startIcon': { m: 0 },
    '& .label': { display: 'none' },
  },
}

function persistGroupBy(by: GroupBy) {
  useSettings.setState({ listGroupBy: by })
  SetListGroupBy(by).catch(reportUnexpected)
}

function TagChips({
  tags,
  selected,
  onTags,
}: {
  tags: readonly string[]
  selected: readonly string[]
  onTags: (tags: string[]) => void
}) {
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, px: 2, py: 1, maxWidth: 320 }}>
      {tags.map((tag) => {
        const on = selected.includes(tag)
        return (
          <Chip
            key={tag}
            size="small"
            label={tag}
            color={on ? 'primary' : 'default'}
            variant={on ? 'filled' : 'outlined'}
            aria-pressed={on}
            onClick={() => onTags(on ? selected.filter((x) => x !== tag) : [...selected, tag])}
          />
        )
      })}
    </Box>
  )
}

function ShowFilterControl({
  filter,
  onFilter,
  tags,
  selectedTags,
  onTags,
}: {
  filter: ModFilter
  onFilter: (filter: ModFilter) => void
  tags: readonly string[]
  selectedTags: readonly string[]
  onTags: (tags: string[]) => void
}) {
  const { t, i18n } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const active = FILTERS.find((item) => item.id === filter)
  const tagged = selectedTags.length > 0
  const showLabel = active ? t`Show: ${active.label(i18n)}` : t`Show`
  const choose = (next: ModFilter) => {
    onFilter(next)
    setAnchor(null)
  }
  return (
    <>
      <Tooltip title={showLabel}>
        <Button
          variant="outlined"
          color={active || tagged ? 'primary' : 'inherit'}
          aria-label={showLabel}
          startIcon={<Filter size={14} />}
          aria-haspopup="menu"
          aria-expanded={anchor !== null}
          onClick={(e) => setAnchor(e.currentTarget)}
          sx={iconWhenCompact}
        >
          <span className="label">{active ? active.label(i18n) : t`Show`}</span>
        </Button>
      </Tooltip>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={() => setAnchor(null)}>
        {[
          { id: 'all' as const, label: t`All mods` },
          ...FILTERS.map((item) => ({ id: item.id, label: item.label(i18n) })),
        ].map((item) => (
          <MenuItem
            key={item.id}
            role="menuitemradio"
            aria-checked={filter === item.id}
            selected={filter === item.id}
            onClick={() => choose(item.id)}
          >
            <ListItemIcon sx={{ color: 'inherit' }}>
              {filter === item.id ? <Check size={16} aria-hidden={true} /> : null}
            </ListItemIcon>
            <ListItemText>{item.label}</ListItemText>
          </MenuItem>
        ))}
        {tags.length > 0 ? <Divider /> : null}
        {tags.length > 0 ? <TagChips tags={tags} selected={selectedTags} onTags={onTags} /> : null}
      </Menu>
    </>
  )
}

function GroupByControl() {
  const { t } = useLingui()
  const by = sanitizeListGroupBy(useSettings((s) => s.listGroupBy))
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [editorOpen, setEditorOpen] = useState(false)
  const tagHint = t`A mod with several tags appears under its first tag.`
  return (
    <>
      <Button
        variant="outlined"
        aria-label={t`Group by`}
        startIcon={<FolderTree size={14} />}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={iconWhenCompact}
      >
        <span className="label">{t`Group by`}</span>
      </Button>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={() => setAnchor(null)}>
        {[
          { id: 'none' as const, label: t`None`, Icon: Ban },
          { id: 'status' as const, label: t`Status`, Icon: ToggleRight },
          { id: 'category' as const, label: t`Category`, Icon: Tag },
          { id: 'source' as const, label: t`Source`, Icon: Library },
          { id: 'tag' as const, label: t`Tag`, Icon: Tag, hint: tagHint },
          { id: 'framework' as const, label: t`Framework`, Icon: Layers },
          { id: 'author' as const, label: t`Author`, Icon: User },
          { id: 'group' as const, label: t`Group`, Icon: FolderTree },
        ].map((item) => (
          <Tooltip key={item.id} title={item.hint ?? ''} placement="right">
            <MenuItem
              role="menuitemradio"
              aria-checked={by === item.id}
              selected={by === item.id}
              onClick={() => {
                persistGroupBy(item.id)
                setAnchor(null)
              }}
            >
              <ListItemIcon sx={{ color: 'inherit' }}>
                <item.Icon size={16} aria-hidden={true} />
              </ListItemIcon>
              <ListItemText>{item.label}</ListItemText>
            </MenuItem>
          </Tooltip>
        ))}
        <Divider />
        <MenuAction
          icon={<Settings2 size={16} aria-hidden={true} />}
          label={t`Edit categories…`}
          onClick={() => {
            setAnchor(null)
            setEditorOpen(true)
          }}
        />
      </Menu>
      <CategoryEditorDialog open={editorOpen} onClose={() => setEditorOpen(false)} />
    </>
  )
}

function BrowseNexus({
  variant,
  toolbar = false,
  size,
  signInFirst = false,
}: {
  variant: 'contained' | 'outlined'
  toolbar?: boolean
  size?: 'large'
  signInFirst?: boolean
}) {
  const { t } = useLingui()
  const signedIn = useNexus((s) => s.signedIn)
  const signIn = signInFirst && !signedIn
  const label = signIn ? t`Sign in to Nexus Mods` : t`Open Nexus Mods`
  return (
    <Button
      variant={variant}
      size={size}
      aria-label={label}
      startIcon={<ExternalLink size={14} />}
      onClick={() => {
        if (signIn) {
          openSettings('nexus')
          return
        }
        openPage(NEXUS).catch(reportUnexpected)
      }}
      sx={toolbar ? iconWhenCompact : undefined}
    >
      <span className="label">{label}</span>
    </Button>
  )
}

function AddArchive({
  variant,
  toolbar = false,
  size,
}: {
  variant: 'contained' | 'outlined'
  toolbar?: boolean
  size?: 'large'
}) {
  const { t } = useLingui()
  const installing = useInstall((s) => s.pending > 0)
  const pick = useInstall((s) => s.pick)
  const locked = useLocked()
  const blocked = installing || locked
  const extraFolder = useSettings((s) => s.games?.stardew?.extraModsFolder ?? '')
  const reason = locked ? t`Stop the game to change mods.` : t`Adding…`
  const add = (
    <DisabledReason title={reason} disabled={blocked}>
      <Button
        variant={variant}
        size={size}
        disabled={blocked}
        aria-label={t`Add archive…`}
        startIcon={installing ? <CircularProgress size={14} color="inherit" /> : <Plus size={14} />}
        onClick={() => {
          pick().catch(reportUnexpected)
        }}
        sx={toolbar ? iconWhenCompact : undefined}
      >
        <span className="label">{installing ? t`Adding…` : t`Add archive…`}</span>
      </Button>
    </DisabledReason>
  )
  return (
    <ExtraFolderMenu
      folder={extraFolder}
      blocked={blocked}
      blockedReason={reason}
      variant={variant}
      size={size}
    >
      {add}
    </ExtraFolderMenu>
  )
}

export function Toolbar({
  query,
  onQuery,
  total,
  filter,
  onFilter,
  tags,
  selectedTags,
  onTags,
}: {
  query: string
  onQuery: (q: string) => void
  total: number
  filter: ModFilter
  onFilter: (filter: ModFilter) => void
  tags: readonly string[]
  selectedTags: readonly string[]
  onTags: (tags: string[]) => void
}) {
  const { t } = useLingui()
  const view = useMods((s) => s.view)
  const setView = useMods((s) => s.setView)
  const narrow = useMediaQuery(compactQuery)
  const [expanded, setExpanded] = useState(false)
  const fieldOpen = searchFieldOpen(narrow, expanded, query)
  const inputRef = useRef<HTMLInputElement>(null)
  const loaded = useMods((s) => s.loaded)
  const placeholder = loaded
    ? t`Filter ${plural(total, { one: '# mod', other: '# mods' })}`
    : t`Filter mods`
  const collapseIfEmpty = () => {
    if (query === '') {
      setExpanded(false)
    }
  }
  useEffect(() => {
    if (expanded) {
      inputRef.current?.focus()
    }
  }, [expanded])
  useEffect(() => onFilterFocus(() => setExpanded(true)), [])
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2, py: 1.25, flexShrink: 0 }}>
      <ViewToggle value={view} onChange={setView} />
      <GroupByControl />
      {fieldOpen ? (
        <SearchField
          value={query}
          onChange={onQuery}
          onBlur={collapseIfEmpty}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              collapseIfEmpty()
            }
          }}
          label={t`Filter mods`}
          placeholder={placeholder}
          inputRef={inputRef}
          sx={{ flex: 1, minWidth: 0 }}
        />
      ) : (
        <Box sx={{ flex: 1, minWidth: 0 }} />
      )}
      <ShowFilterControl
        filter={filter}
        onFilter={onFilter}
        tags={tags}
        selectedTags={selectedTags}
        onTags={onTags}
      />
      {narrow ? (
        <Button
          variant="outlined"
          aria-label={query === '' ? t`Filter mods` : t`Filter mods, filter active`}
          aria-expanded={fieldOpen}
          onClick={() => setExpanded(true)}
          sx={{ minWidth: 36, px: 0, position: 'relative' }}
        >
          <Filter size={14} />
          {query === '' ? null : (
            <Box
              aria-hidden={true}
              sx={{
                position: 'absolute',
                top: 6,
                right: 6,
                width: 6,
                height: 6,
                borderRadius: '50%',
                bgcolor: 'primary.main',
              }}
            />
          )}
        </Button>
      ) : null}
      <BrowseNexus variant="outlined" toolbar={true} />
      <AddArchive variant="outlined" toolbar={true} />
    </Box>
  )
}

export function EmptyMods({ profileId }: { profileId: string }) {
  const { t } = useLingui()
  return (
    <EmptyState
      icon={<Download />}
      title={t`No mods yet`}
      action={
        <>
          <TipBanner tip="mods">{t`Drop archives anywhere on the window, or Open Nexus Mods to find mods.`}</TipBanner>
          <Box sx={{ display: 'flex', gap: 1.5 }}>
            <BrowseNexus variant="contained" size="large" signInFirst={true} />
            <AddArchive variant="outlined" size="large" />
          </Box>
          <Button
            variant="text"
            onClick={() => openImport({ profileId })}
            sx={{ textDecoration: 'underline' }}
          >
            {t`Or import a shared profile…`}
          </Button>
          <Button variant="text" onClick={openDownloadsDialog} sx={{ textDecoration: 'underline' }}>
            {t`Or add from the downloads folder…`}
          </Button>
        </>
      }
    >
      {t`Paste a share link, a collection link, or (Premium) a Nexus mod link with Ctrl+V.`}
    </EmptyState>
  )
}

export type ModFilter = 'all' | 'disabled' | 'update' | 'problem' | 'pinned' | 'local' | 'recent'
