import type { I18n } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ButtonBase,
  CircularProgress,
  Divider,
  InputAdornment,
  ListItemIcon,
  ListItemText,
  ListSubheader,
  Menu,
  MenuItem,
  TextField,
  Tooltip,
  useMediaQuery,
} from '@mui/material'
import { Browser, Clipboard } from '@wailsio/runtime'
import {
  Ban,
  Check,
  Copy,
  Download,
  ExternalLink,
  Filter,
  FolderTree,
  Layers,
  LayoutGrid,
  Library,
  List,
  Plus,
  Search,
  Settings2,
  Tag,
  ToggleRight,
  User,
} from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { SetListGroupBy } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { compact, compactQuery, searchFieldOpen } from '../game/compact.ts'
import { useInstall } from '../install/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { formatDiscord } from '../share/modList.ts'
import { openImport } from '../share/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { TipBanner } from '../tips/TipBanner.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { CategoryEditorDialog } from './CategoryEditor.tsx'
import { onFilterFocus } from './filterFocus.ts'
import { type GroupBy, sanitizeListGroupBy } from './group.ts'
import { nexusIdOf } from './lookup.ts'
import { formatModList, type ModListFormat } from './modListText.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const NEXUS = 'https://www.nexusmods.com/stardewvalley/mods'
const FILTERS: readonly {
  id: Exclude<ModFilter, 'all'>
  label: (i18n: I18n) => string
}[] = [
  { id: 'disabled', label: (i18n) => i18n._(msg`Disabled`) },
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

const viewButton = (active: boolean) => ({
  width: 34,
  height: 30,
  borderRadius: '6px',
  bgcolor: active ? 'rgba(255,255,255,0.16)' : 'transparent',
  color: active ? '#ffffff' : 'text.secondary',
  '&:hover': { bgcolor: active ? 'rgba(255,255,255,0.16)' : 'rgba(255,255,255,0.08)' },
})

function persistGroupBy(by: GroupBy) {
  useSettings.setState({ listGroupBy: by })
  SetListGroupBy(by).catch(reportUnexpected)
}

function ShowFilterControl({
  filter,
  onFilter,
}: {
  filter: ModFilter
  onFilter: (filter: ModFilter) => void
}) {
  const { t, i18n } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const active = FILTERS.find((item) => item.id === filter)
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
          color={active ? 'primary' : 'inherit'}
          aria-label={showLabel}
          startIcon={<Filter size={14} />}
          onClick={(e) => setAnchor(e.currentTarget)}
          sx={iconWhenCompact}
        >
          <span className="label">{active ? active.label(i18n) : t`Show`}</span>
        </Button>
      </Tooltip>
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        transitionDuration={0}
      >
        {[
          { id: 'all' as const, label: t`All mods` },
          ...FILTERS.map((item) => ({ id: item.id, label: item.label(i18n) })),
        ].map((item) => (
          <MenuItem key={item.id} selected={filter === item.id} onClick={() => choose(item.id)}>
            <ListItemIcon sx={{ color: 'inherit' }}>
              {filter === item.id ? <Check size={16} aria-hidden={true} /> : null}
            </ListItemIcon>
            <ListItemText>{item.label}</ListItemText>
          </MenuItem>
        ))}
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
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={iconWhenCompact}
      >
        <span className="label">{t`Group by`}</span>
      </Button>
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        transitionDuration={0}
      >
        {[
          { id: 'none' as const, label: t`None`, Icon: Ban },
          { id: 'status' as const, label: t`Status`, Icon: ToggleRight },
          { id: 'category' as const, label: t`Category`, Icon: Tag },
          { id: 'source' as const, label: t`Source`, Icon: Library },
          { id: 'tag' as const, label: t`Tag`, Icon: Tag, hint: tagHint },
          { id: 'framework' as const, label: t`Framework`, Icon: Layers },
          { id: 'author' as const, label: t`Author`, Icon: User },
        ].map((item) => (
          <Tooltip key={item.id} title={item.hint ?? ''} placement="right">
            <MenuItem
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
        <MenuItem
          onClick={() => {
            setAnchor(null)
            setEditorOpen(true)
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <Settings2 size={16} aria-hidden={true} />
          </ListItemIcon>
          <ListItemText>{t`Edit categories…`}</ListItemText>
        </MenuItem>
      </Menu>
      <CategoryEditorDialog open={editorOpen} onClose={() => setEditorOpen(false)} />
    </>
  )
}

function BrowseNexus({
  variant,
  toolbar = false,
  size,
}: {
  variant: 'contained' | 'outlined'
  toolbar?: boolean
  size?: 'large'
}) {
  const { t } = useLingui()
  const label = toolbar ? t`Open Nexus` : t`Browse Nexus`
  return (
    <Button
      variant={variant}
      size={size}
      aria-label={label}
      startIcon={<ExternalLink size={14} />}
      onClick={() => {
        Browser.OpenURL(NEXUS).catch(reportUnexpected)
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
  return (
    <DisabledReason
      title={locked ? t`Stop the game to change mods.` : t`Adding…`}
      disabled={blocked}
    >
      <Button
        variant={variant}
        size={size}
        disabled={blocked}
        aria-label={t`Add archive`}
        startIcon={installing ? <CircularProgress size={14} color="inherit" /> : <Plus size={14} />}
        onClick={() => {
          pick().catch(reportUnexpected)
        }}
        sx={toolbar ? iconWhenCompact : undefined}
      >
        <span className="label">{installing ? t`Adding…` : t`Add archive`}</span>
      </Button>
    </DisabledReason>
  )
}

function CopyModListControl() {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const enabled = mods.some((mod) => mod.enabled)
  const copy = (format: ModListFormat | 'discord') => {
    setAnchor(null)
    if (!profile) {
      return
    }
    const items = mods.map((mod) => {
      const nexusId = nexusIdOf(profile, mod)
      const url = nexusId > 0 ? `https://www.nexusmods.com/stardewvalley/mods/${nexusId}` : ''
      return {
        enabled: mod.enabled,
        name: mod.name,
        version: mod.version,
        url,
        ...(nexusId > 0 ? { nexusUrl: url } : {}),
      }
    })
    const text =
      format === 'discord' ? formatDiscord(items).join('\n\n') : formatModList(items, format)
    Clipboard.SetText(text).then(
      () => useToasts.getState().push({ kind: 'success', title: t`Mod list copied` }),
      reportUnexpected,
    )
  }
  const enableFirst = t`Enable a mod first.`
  return (
    <>
      <DisabledReason title={enableFirst} disabled={!enabled}>
        <Button
          variant="outlined"
          aria-label={t`Copy mod list`}
          disabled={!enabled}
          startIcon={<Copy size={14} />}
          onClick={(e) => setAnchor(e.currentTarget)}
          sx={iconWhenCompact}
        >
          <span className="label">{t`Copy`}</span>
        </Button>
      </DisabledReason>
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        transitionDuration={0}
      >
        <ListSubheader sx={{ lineHeight: '32px', bgcolor: 'transparent' }}>
          {t`Copy mod list`}
        </ListSubheader>
        <MenuItem disabled={!enabled} onClick={() => copy('markdown')}>
          <ListItemText>{t`Markdown`}</ListItemText>
        </MenuItem>
        <MenuItem disabled={!enabled} onClick={() => copy('plain')}>
          <ListItemText>{t`Plain text`}</ListItemText>
        </MenuItem>
        <MenuItem disabled={!enabled} onClick={() => copy('discord')}>
          <ListItemText>{t`Discord`}</ListItemText>
        </MenuItem>
      </Menu>
    </>
  )
}

export function Toolbar({
  query,
  onQuery,
  total,
  filter,
  onFilter,
}: {
  query: string
  onQuery: (q: string) => void
  total: number
  filter: ModFilter
  onFilter: (filter: ModFilter) => void
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
      <Box
        role="group"
        aria-label={t`View`}
        sx={{
          display: 'flex',
          p: '3px',
          gap: '2px',
          bgcolor: 'rgba(0,0,0,0.3)',
          borderRadius: '8px',
        }}
      >
        <ButtonBase
          aria-label={t`Grid view`}
          aria-pressed={view === 'grid'}
          onClick={() => setView('grid')}
          sx={viewButton(view === 'grid')}
        >
          <LayoutGrid size={15} />
        </ButtonBase>
        <ButtonBase
          aria-label={t`List view`}
          aria-pressed={view === 'list'}
          onClick={() => setView('list')}
          sx={viewButton(view === 'list')}
        >
          <List size={15} />
        </ButtonBase>
      </Box>
      <GroupByControl />
      {fieldOpen ? (
        <TextField
          size="small"
          value={query}
          onChange={(e) => onQuery(e.target.value)}
          onBlur={collapseIfEmpty}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              collapseIfEmpty()
            }
          }}
          placeholder={placeholder}
          inputRef={inputRef}
          slotProps={{
            htmlInput: { id: 'mods-filter', 'aria-label': t`Filter mods` },
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <Search size={14} />
                </InputAdornment>
              ),
              sx: {
                height: 36,
                fontSize: 13,
                borderRadius: '6px',
                bgcolor: 'rgba(0,0,0,0.30)',
                '& .MuiOutlinedInput-notchedOutline': { borderColor: 'rgba(255,255,255,0.15)' },
              },
            },
          }}
          sx={{ flex: 1, minWidth: 0 }}
        />
      ) : (
        <Box sx={{ flex: 1, minWidth: 0 }} />
      )}
      <ShowFilterControl filter={filter} onFilter={onFilter} />
      {narrow ? (
        <Button
          variant="outlined"
          aria-label={t`Filter mods`}
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
      <CopyModListControl />
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
          <TipBanner tip="mods">{t`Drop archives anywhere on the window, or Browse Nexus to find mods.`}</TipBanner>
          <Box sx={{ display: 'flex', gap: 1.5 }}>
            <BrowseNexus variant="contained" size="large" />
            <AddArchive variant="outlined" size="large" />
          </Box>
          <Button
            variant="text"
            onClick={() => openImport({ profileId })}
            sx={{ textDecoration: 'underline' }}
          >
            {t`Or import a shared profile`}
          </Button>
        </>
      }
    >
      {t`Paste a share link, a collection link, or (Premium) a Nexus mod link with Ctrl+V.`}
    </EmptyState>
  )
}

export type ModFilter = 'all' | 'disabled' | 'update' | 'problem' | 'pinned' | 'local' | 'recent'
