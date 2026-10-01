import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItemButton,
  ListItemText,
  Typography,
} from '@mui/material'
import { ChevronDown, ChevronRight, File, Folder } from 'lucide-react'
import { useState } from 'react'
import type { RemapNode } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { paper } from '../mods/paper.ts'
import { useInstall } from './store.ts'

const bytesInKb = 1024

function formatSize(n: number): string {
  if (n < bytesInKb) {
    return `${n} B`
  }
  if (n < bytesInKb * bytesInKb) {
    return `${(n / bytesInKb).toFixed(1)} KB`
  }
  return `${(n / (bytesInKb * bytesInKb)).toFixed(1)} MB`
}

function NodeRow({
  node,
  selected,
  onSelect,
}: {
  node: RemapNode
  selected: string
  onSelect: (path: string, dir: boolean) => void
}) {
  const [open, setOpen] = useState(true)
  if (!node.dir) {
    return (
      <ListItemButton disabled={true} sx={{ py: 0.25, pl: 4 }}>
        <Box component={File} size={14} strokeWidth={1.75} sx={{ mr: 1, flexShrink: 0 }} />
        <ListItemText
          primary={node.name}
          secondary={formatSize(node.size)}
          slotProps={{ primary: { noWrap: true }, secondary: { noWrap: true } }}
        />
      </ListItemButton>
    )
  }
  return (
    <>
      <ListItemButton
        selected={selected === node.path}
        onClick={() => onSelect(node.path, true)}
        sx={{ py: 0.25 }}
      >
        <Box
          component="span"
          onClick={(ev) => {
            ev.stopPropagation()
            setOpen((v) => !v)
          }}
          sx={{ display: 'flex', mr: 0.5 }}
        >
          {open ? (
            <ChevronDown size={14} strokeWidth={1.75} />
          ) : (
            <ChevronRight size={14} strokeWidth={1.75} />
          )}
        </Box>
        <Box component={Folder} size={14} strokeWidth={1.75} sx={{ mr: 1, flexShrink: 0 }} />
        <ListItemText
          primary={node.name}
          secondary={formatSize(node.size)}
          slotProps={{ primary: { noWrap: true }, secondary: { noWrap: true } }}
        />
      </ListItemButton>
      <Collapse in={open} timeout={0}>
        <List disablePadding={true} sx={{ pl: 2 }}>
          {(node.children ?? []).map((c) => (
            <NodeRow key={c.path} node={c} selected={selected} onSelect={onSelect} />
          ))}
        </List>
      </Collapse>
    </>
  )
}

function RemapBody() {
  const { t } = useLingui()
  const session = useInstall((s) => s.remap)
  const close = useInstall((s) => s.closeRemap)
  const choose = useInstall((s) => s.chooseRoot)
  const [selected, setSelected] = useState('')
  const [dir, setDir] = useState(false)
  if (!session) {
    return null
  }
  return (
    <Dialog
      open={true}
      onClose={close}
      slotProps={{ paper }}
      transitionDuration={0}
      sx={{ '& .MuiDialog-paper': { minWidth: 420, maxHeight: '80vh' } }}
    >
      <DialogTitle>{t`Choose the mod folder`}</DialogTitle>
      <DialogContent>
        <Typography variant="body2" sx={{ mb: 1 }}>
          {t`This archive has no SMAPI mod where Mortar expects one. Pick the folder that holds manifest.json.`}
        </Typography>
        <List dense={true} disablePadding={true}>
          {(session.ask.tree ?? []).map((n) => (
            <NodeRow
              key={n.path}
              node={n}
              selected={selected}
              onSelect={(path, isDir) => {
                setSelected(path)
                setDir(isDir)
              }}
            />
          ))}
        </List>
      </DialogContent>
      <DialogActions>
        <Button onClick={close} sx={{ whiteSpace: 'nowrap' }}>
          {t`Cancel`}
        </Button>
        <Button
          variant="contained"
          disabled={!dir || selected === ''}
          onClick={() => choose(selected)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Use this folder`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export function RemapDialog() {
  const session = useInstall((s) => s.remap)
  if (!session) {
    return null
  }
  return <RemapBody key={session.key} />
}
