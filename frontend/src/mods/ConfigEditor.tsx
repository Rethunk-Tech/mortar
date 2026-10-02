import { useLingui } from '@lingui/react/macro'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { ChevronDown, Pencil } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  ReadConfig,
  WriteConfig,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { type ConfigNode, parseConfig, setAt, stringifyConfig } from './configForm.ts'
import { paper } from './paper.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const text = { fontSize: 13 } as const
const noWrap = { whiteSpace: 'nowrap' } as const
const row = { display: 'flex', alignItems: 'center', gap: 1, minHeight: 36, ...text } as const
const numberPattern = /^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?$/

function openTarget() {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, id: openId } : null
}

function Field({
  label,
  node,
  path,
  onChange,
  onOpen,
}: {
  label: string
  node: ConfigNode
  path: readonly number[]
  onChange: (path: readonly number[], next: ConfigNode) => void
  onOpen: () => void
}) {
  const { t } = useLingui()
  const [draft, setDraft] = useState('')
  switch (node.kind) {
    case 'bool':
      return (
        <Box sx={row}>
          <Typography sx={{ flex: 1, minWidth: 0, ...text }}>{label}</Typography>
          <Switch
            size="small"
            checked={node.value}
            onChange={(_, checked) => onChange(path, { kind: 'bool', value: checked })}
          />
        </Box>
      )
    case 'int':
    case 'float':
      return (
        <TextField
          size="small"
          fullWidth={true}
          label={label}
          type="number"
          value={String(node.value)}
          onChange={(e) => {
            const raw = e.target.value
            if (!numberPattern.test(raw)) {
              return
            }
            const float = node.kind === 'float' || raw.includes('.')
            onChange(path, float ? { kind: 'float', value: raw } : { kind: 'int', value: raw })
          }}
        />
      )
    case 'string':
      return (
        <TextField
          size="small"
          fullWidth={true}
          label={label}
          value={node.value}
          onChange={(e) => onChange(path, { kind: 'string', value: e.target.value })}
        />
      )
    case 'strings':
      return (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          <Typography sx={text}>{label}</Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
            {node.value.map((chip, i) => (
              <Chip
                key={`${chip}-${String(i)}`}
                size="small"
                label={chip}
                onDelete={() =>
                  onChange(path, { kind: 'strings', value: node.value.filter((_, j) => j !== i) })
                }
              />
            ))}
          </Box>
          <TextField
            size="small"
            value={draft}
            label={t`Add a value`}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key !== 'Enter') {
                return
              }
              e.preventDefault()
              const next = draft.trim()
              if (!next) {
                return
              }
              onChange(path, { kind: 'strings', value: [...node.value, next] })
              setDraft('')
            }}
          />
        </Box>
      )
    case 'object':
      return (
        <Accordion
          disableGutters={true}
          elevation={0}
          sx={{ bgcolor: 'transparent', '&:before': { display: 'none' } }}
        >
          <AccordionSummary expandIcon={<ChevronDown size={16} />}>
            <Typography sx={text}>{label}</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ display: 'flex', flexDirection: 'column', gap: 1, pl: 1 }}>
            {node.entries.map((entry, i) => (
              <Field
                key={entry.key}
                label={entry.key}
                node={entry.node}
                path={[...path, i]}
                onChange={onChange}
                onOpen={onOpen}
              />
            ))}
          </AccordionDetails>
        </Accordion>
      )
    case 'readonly':
      return (
        <Box sx={row}>
          <Typography sx={{ flex: 1, minWidth: 0, ...text }}>
            {t`${label}: ${node.json}`}
          </Typography>
          <Button size="small" variant="outlined" onClick={onOpen} sx={noWrap}>
            {t`Open in editor`}
          </Button>
        </Box>
      )
    default: {
      const exhaustive: never = node
      return exhaustive
    }
  }
}

function Fields({
  tree,
  onChange,
  onOpen,
}: {
  tree: ConfigNode | null
  onChange: (path: readonly number[], next: ConfigNode) => void
  onOpen: () => void
}) {
  const { t } = useLingui()
  if (tree?.kind === 'object') {
    return tree.entries.map((entry, i) => (
      <Field
        key={entry.key}
        label={entry.key}
        node={entry.node}
        path={[i]}
        onChange={onChange}
        onOpen={onOpen}
      />
    ))
  }
  if (tree) {
    return (
      <Field label={t`config.json`} node={tree} path={[]} onChange={onChange} onOpen={onOpen} />
    )
  }
  return null
}

export function ConfigEditor({
  mod,
  open,
  onClose,
}: {
  mod: Mod
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const openConfig = useMods((s) => s.openConfig)
  const [tree, setTree] = useState<ConfigNode | null>(null)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [discardOpen, setDiscardOpen] = useState(false)
  const readGen = useRef(0)
  useEffect(() => {
    if (!open) {
      readGen.current += 1
      setTree(null)
      setError('')
      setSaved(false)
      setDirty(false)
      return
    }
    const target = openTarget()
    if (!target) {
      return
    }
    readGen.current += 1
    const seq = readGen.current
    setSaved(false)
    setDirty(false)
    ReadConfig(target.game, target.id, mod.key, mod.uniqueId)
      .then((raw) => {
        if (seq !== readGen.current) {
          return
        }
        setError('')
        setTree(parseConfig(raw))
      })
      .catch((e: unknown) => {
        if (seq !== readGen.current) {
          return
        }
        setTree(null)
        setError(e instanceof Error ? e.message : String(e))
      })
  }, [open, mod.key, mod.uniqueId])
  const save = () => {
    const seq = readGen.current
    const target = openTarget()
    if (!(target && tree)) {
      return
    }
    WriteConfig(target.game, target.id, mod.key, mod.uniqueId, stringifyConfig(tree))
      .then(() => {
        if (seq === readGen.current) {
          setSaved(true)
          setDirty(false)
        }
      })
      .catch(reportUnexpected)
  }
  const close = () => {
    if (dirty) {
      setDiscardOpen(true)
      return
    }
    onClose()
  }
  return (
    <>
      <Dialog
        open={open}
        onClose={close}
        fullWidth={true}
        maxWidth="sm"
        transitionDuration={0}
        slotProps={{ paper }}
      >
        <DialogTitle>{t`Edit config.json`}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          {error ? <Typography sx={text}>{error}</Typography> : null}
          <Fields
            tree={tree}
            onChange={(path, next) => {
              setSaved(false)
              setDirty(true)
              setTree((cur) => (cur ? setAt(cur, path, next) : cur))
            }}
            onOpen={() => openConfig(mod).catch(reportUnexpected)}
          />
        </DialogContent>
        <DialogActions>
          {saved ? <Typography sx={{ mr: 'auto', ...text }}>{t`Saved`}</Typography> : null}
          <Button onClick={close} sx={noWrap}>{t`Close`}</Button>
          <Button onClick={save} disabled={locked || !tree} sx={noWrap}>
            {t`Save`}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={discardOpen} onClose={() => setDiscardOpen(false)}>
        <DialogTitle>{t`Discard changes?`}</DialogTitle>
        <DialogActions>
          <Button onClick={() => setDiscardOpen(false)}>{t`Cancel`}</Button>
          <Button onClick={onClose} autoFocus={true}>{t`Discard`}</Button>
        </DialogActions>
      </Dialog>
    </>
  )
}

export function EditConfigButton({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button
        size="small"
        variant="outlined"
        startIcon={<Pencil size={14} />}
        onClick={() => setOpen(true)}
        sx={noWrap}
      >
        {t`Edit`}
      </Button>
      <ConfigEditor mod={mod} open={open} onClose={() => setOpen(false)} />
    </>
  )
}
