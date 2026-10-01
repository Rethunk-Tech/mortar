import { plural } from '@lingui/core/macro'
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
  Menu,
  MenuItem,
  TextField,
  Tooltip,
  Typography,
  useMediaQuery,
} from '@mui/material'
import { Browser } from '@wailsio/runtime'
import {
  Ban,
  Download,
  ExternalLink,
  Filter,
  FolderTree,
  LayoutGrid,
  Library,
  List,
  Plus,
  Search,
  Settings2,
  Tag,
  User,
} from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { SetListGroupBy } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { compact, compactQuery, searchFieldOpen } from '../game/compact.ts'
import { useInstall } from '../install/store.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { TipBanner } from '../tips/TipBanner.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { CategoryEditorDialog } from './CategoryEditor.tsx'
import { onFilterFocus } from './filterFocus.ts'
import { type GroupBy, sanitizeListGroupBy } from './group.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const NEXUS = 'https://www.nexusmods.com/stardewvalley/mods'

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
          { id: 'status' as const, label: t`Status`, Icon: FolderTree },
          { id: 'category' as const, label: t`Category`, Icon: FolderTree },
          { id: 'source' as const, label: t`Source`, Icon: Library },
          { id: 'tag' as const, label: t`Tag`, Icon: Tag, hint: tagHint },
          { id: 'framework' as const, label: t`Framework`, Icon: FolderTree },
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
              <ListItemIcon>
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
          <ListItemIcon>
            <Settings2 size={16} aria-hidden={true} />
          </ListItemIcon>
          <ListItemText>{t`Edit categories…`}</ListItemText>
        </MenuItem>
      </Menu>
      <CategoryEditorDialog open={editorOpen} onClose={() => setEditorOpen(false)} />
    </>
  )
}

export function BrowseNexus({
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

export function AddArchive({
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
  return (
    <Button
      variant={variant}
      size={size}
      disabled={installing || locked}
      aria-label={t`Add archive`}
      startIcon={installing ? <CircularProgress size={14} color="inherit" /> : <Plus size={14} />}
      onClick={() => {
        pick().catch(reportUnexpected)
      }}
      sx={toolbar ? iconWhenCompact : undefined}
    >
      <span className="label">{installing ? t`Adding…` : t`Add archive`}</span>
    </Button>
  )
}

export function Toolbar({
  query,
  onQuery,
  total,
  configurableOnly,
  onConfigurable,
}: {
  query: string
  onQuery: (q: string) => void
  total: number
  configurableOnly: boolean
  onConfigurable: (on: boolean) => void
}) {
  const { t } = useLingui()
  const view = useMods((s) => s.view)
  const setView = useMods((s) => s.setView)
  const narrow = useMediaQuery(compactQuery)
  const [expanded, setExpanded] = useState(false)
  const fieldOpen = searchFieldOpen(narrow, expanded, query)
  const inputRef = useRef<HTMLInputElement>(null)
  const placeholder = t`Filter ${plural(total, { one: '# mod', other: '# mods' })}`
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
      <Button
        variant="outlined"
        aria-label={t`Configurable`}
        aria-pressed={configurableOnly}
        startIcon={<Settings2 size={14} />}
        onClick={() => onConfigurable(!configurableOnly)}
        sx={{
          ...iconWhenCompact,
          bgcolor: configurableOnly ? 'rgba(255,255,255,0.16)' : undefined,
        }}
      >
        <span className="label">{t`Configurable`}</span>
      </Button>
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
      <BrowseNexus variant="outlined" toolbar={true} />
      <AddArchive variant="outlined" toolbar={true} />
    </Box>
  )
}

export function EmptyMods({ profileId }: { profileId: string }) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 2.25,
        px: 3,
        textAlign: 'center',
      }}
    >
      <Typography sx={{ fontSize: 26, fontWeight: 700 }}>{t`No mods yet`}</Typography>
      <TipBanner tip="mods">
        {t`Drop archives anywhere on the window, or Browse Nexus to find mods.`}
      </TipBanner>
      <Typography sx={{ maxWidth: 520, fontSize: 15, lineHeight: 1.5 }}>
        {t`Add mods from an archive you downloaded, or find them on Nexus.`}
      </Typography>
      <Box sx={{ display: 'flex', gap: 1.5 }}>
        <BrowseNexus variant="contained" size="large" />
        <AddArchive variant="outlined" size="large" />
      </Box>
      <Box
        sx={{
          mt: 1.5,
          px: 1.75,
          py: 1.25,
          display: 'flex',
          alignItems: 'center',
          gap: 1.25,
          fontSize: 14,
          border: '1px dashed rgba(255,255,255,0.25)',
          borderRadius: '8px',
        }}
      >
        <Download size={16} aria-hidden={true} />
        {t`You can also drop archives anywhere on the window.`}
      </Box>
      <Button
        variant="text"
        onClick={() => openImport({ profileId })}
        sx={{ textDecoration: 'underline' }}
      >
        {t`Or import a shared profile`}
      </Button>
    </Box>
  )
}
