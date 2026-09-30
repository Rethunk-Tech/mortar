import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ButtonBase,
  CircularProgress,
  InputAdornment,
  TextField,
  useMediaQuery,
} from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { ExternalLink, Filter, LayoutGrid, List, Plus, Search } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { compact, compactQuery, searchFieldOpen } from '../game/compact.ts'
import { useInstall } from '../install/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { onFilterFocus } from './filterFocus.ts'
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
}: {
  query: string
  onQuery: (q: string) => void
  total: number
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
