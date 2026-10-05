import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItemButton,
  ListItemText,
  Typography,
} from '@mui/material'
import { Folder } from 'lucide-react'
import { useState } from 'react'
import type { RemapNode } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { chooseOverlay } from './overlayPlace.ts'
import { useInstall } from './store.ts'

interface FolderOption {
  path: string
  name: string
  depth: number
}

function folders(nodes: readonly RemapNode[], depth = 1): FolderOption[] {
  return nodes
    .filter((n) => n.dir)
    .flatMap((n) => [
      { path: n.path, name: n.name, depth },
      ...folders(n.children ?? [], depth + 1),
    ])
}

function FolderPick({
  label,
  root,
  nodes,
  selected,
  onSelect,
}: {
  label: string
  root: string
  nodes: readonly RemapNode[]
  selected: string
  onSelect: (path: string) => void
}) {
  const options = [{ path: '', name: root, depth: 0 }, ...folders(nodes)]
  return (
    <Box sx={{ minWidth: 0 }}>
      <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
        {label}
      </Typography>
      <List
        dense={true}
        disablePadding={true}
        aria-label={label}
        sx={{ maxHeight: 320, overflowY: 'auto' }}
      >
        {options.map((o) => (
          <ListItemButton
            key={o.path || '.'}
            selected={selected === o.path}
            onClick={() => onSelect(o.path)}
            sx={{ py: 0.25, pl: 1 + o.depth * 2 }}
          >
            <Box component={Folder} size={14} strokeWidth={1.75} sx={{ mr: 1, flexShrink: 0 }} />
            <ListItemText primary={o.name} slotProps={{ primary: { noWrap: true } }} />
          </ListItemButton>
        ))}
      </List>
    </Box>
  )
}

/** Where an optional file without a manifest goes inside its main mod, when Mortar cannot tell from its folders. */
export function OverlayPlaceDialog() {
  const { t } = useLingui()
  const session = useInstall((s) => s.remap)
  const close = useInstall((s) => s.closeRemap)
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const overlay = session?.ask.overlay
  if (!(session && overlay)) {
    return null
  }
  const file = session.source.name || session.key
  return (
    <Dialog
      open={true}
      onClose={close}
      sx={{ '& .MuiDialog-paper': { minWidth: 560, maxHeight: '80vh' } }}
    >
      <DialogTitle>{t`Place ${file}`}</DialogTitle>
      <DialogContent>
        <Typography variant="body2" sx={{ mb: 1.5 }}>
          {t`This optional file has no manifest and replaces files of ${overlay.baseLabel}. Mortar could not tell where they go: pick the folder to take from it and the main mod folder it goes into.`}
        </Typography>
        <Box sx={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr)', gap: 2 }}>
          <FolderPick
            label={t`From this file`}
            root={t`Whole file`}
            nodes={session.ask.tree ?? []}
            selected={from}
            onSelect={setFrom}
          />
          <FolderPick
            label={t`Into ${{ profile: overlay.baseLabel }}`}
            root={t`Main mod's folder`}
            nodes={overlay.targets ?? []}
            selected={to}
            onSelect={setTo}
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={close}>{t`Cancel`}</Button>
        <Button
          variant="contained"
          onClick={() => {
            chooseOverlay(from, to).catch(reportUnexpected)
          }}
        >
          {t`Place here`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
