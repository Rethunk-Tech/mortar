import { Trans, useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  CircularProgress,
  InputAdornment,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { ExternalLink, FilePlus, LayoutGrid, List, Search } from 'lucide-react'
import { useInstall } from '../install/store.ts'
import { useMods, type View } from './store.ts'

const NEXUS = 'https://www.nexusmods.com/stardewvalley/mods'

export function BrowseNexus({ variant }: { variant: 'contained' | 'outlined' }) {
  return (
    <Button
      variant={variant}
      startIcon={<ExternalLink size={16} />}
      onClick={() => void Browser.OpenURL(NEXUS)}
    >
      <Trans>Browse Nexus</Trans>
    </Button>
  )
}

export function AddArchive({ variant }: { variant: 'contained' | 'outlined' }) {
  const installing = useInstall((s) => s.pending > 0)
  const pick = useInstall((s) => s.pick)
  return (
    <Button
      variant={variant}
      disabled={installing}
      startIcon={
        installing ? <CircularProgress size={16} color="inherit" /> : <FilePlus size={16} />
      }
      onClick={() => void pick()}
      sx={{ whiteSpace: 'nowrap' }}
    >
      {installing ? <Trans>Adding…</Trans> : <Trans>Add archive</Trans>}
    </Button>
  )
}

export function Toolbar({ query, onQuery }: { query: string; onQuery: (q: string) => void }) {
  const { t } = useLingui()
  const view = useMods((s) => s.view)
  const setView = useMods((s) => s.setView)
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 1.5, mb: 2 }}>
      <TextField
        size="small"
        value={query}
        onChange={(e) => onQuery(e.target.value)}
        placeholder={t`Search mods`}
        slotProps={{
          htmlInput: { 'aria-label': t`Search mods` },
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <Search size={16} />
              </InputAdornment>
            ),
          },
        }}
        sx={{ flex: '1 1 200px', maxWidth: 360 }}
      />
      <ToggleButtonGroup
        exclusive={true}
        size="small"
        value={view}
        onChange={(_, v: View | null) => v && setView(v)}
      >
        <ToggleButton value="grid" aria-label={t`Grid view`}>
          <LayoutGrid size={16} />
        </ToggleButton>
        <ToggleButton value="list" aria-label={t`List view`}>
          <List size={16} />
        </ToggleButton>
      </ToggleButtonGroup>
      <Box sx={{ flex: 1 }} />
      <AddArchive variant="outlined" />
      <BrowseNexus variant="outlined" />
    </Box>
  )
}
