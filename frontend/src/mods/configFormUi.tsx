import { useLingui } from '@lingui/react/macro'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  FormHelperText,
  IconButton,
  MenuItem,
  Select,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { ChevronDown, ChevronUp, Plus, Trash2 } from 'lucide-react'
import type { ReactNode } from 'react'
import { fieldLabel } from './configFields.ts'
import { type ConfigNode, emptyItem, isKeybind } from './configForm.ts'

const text = { fontSize: 13 } as const
const noWrap = { whiteSpace: 'nowrap' } as const
const row = { display: 'flex', alignItems: 'center', gap: 1, minHeight: 36, ...text } as const
const numberPattern = /^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?$/

interface FieldProps {
  label: string
  node: ConfigNode
  path: readonly number[]
  onChange: (path: readonly number[], next: ConfigNode) => void
  onOpen: () => void
}

function hintText(node: ConfigNode): string {
  if (node.kind === 'choice') {
    return node.description
  }
  if (hasHint(node)) {
    return node.hint?.description ?? ''
  }
  return ''
}

function defaultOf(node: ConfigNode): string {
  if (node.kind === 'choice') {
    return node.defaultValue
  }
  if (hasHint(node)) {
    return node.hint?.defaultValue ?? ''
  }
  return ''
}

function hasHint(
  node: ConfigNode,
): node is Extract<ConfigNode, { kind: 'bool' | 'int' | 'float' | 'string' | 'keybind' }> {
  return (
    node.kind === 'bool' ||
    node.kind === 'int' ||
    node.kind === 'float' ||
    node.kind === 'string' ||
    node.kind === 'keybind'
  )
}

function sectionOf(node: ConfigNode): string {
  if (node.kind === 'choice') {
    return node.section
  }
  if (hasHint(node)) {
    return node.hint?.section ?? ''
  }
  return ''
}

function Help({ node }: { node: ConfigNode }) {
  const { t } = useLingui()
  const fallback = defaultOf(node)
  const help = hintText(node)
  return (
    <>
      {fallback ? <FormHelperText>{t`Default: ${fallback}`}</FormHelperText> : null}
      {help ? <FormHelperText>{help}</FormHelperText> : null}
    </>
  )
}

function Reset({ node, path, onChange }: Omit<FieldProps, 'label' | 'onOpen'>) {
  const { t } = useLingui()
  const fallback = defaultOf(node)
  if (!fallback) {
    return null
  }
  return (
    <Button
      size="small"
      onClick={() => {
        if (node.kind === 'choice' || hasHint(node)) {
          if (node.kind === 'bool') {
            onChange(path, { ...node, value: fallback === 'true' })
            return
          }
          onChange(path, { ...node, value: fallback })
        }
      }}
      sx={noWrap}
    >
      {t`Reset`}
    </Button>
  )
}

function BoolField(p: FieldProps) {
  const node = p.node
  if (node.kind !== 'bool') {
    return null
  }
  return (
    <Box>
      <Box sx={row}>
        <Typography sx={{ flex: 1, minWidth: 0, ...text }}>{p.label}</Typography>
        <Switch
          size="small"
          checked={node.value}
          onChange={(_, checked) => p.onChange(p.path, { ...node, value: checked })}
        />
        <Reset {...p} />
      </Box>
      <Help node={p.node} />
    </Box>
  )
}

function NumberField(p: FieldProps) {
  if (p.node.kind !== 'int' && p.node.kind !== 'float') {
    return null
  }
  return (
    <Box>
      <TextField
        size="small"
        fullWidth={true}
        label={p.label}
        type="number"
        value={String(p.node.value)}
        onChange={(e) => {
          const raw = e.target.value
          if (!numberPattern.test(raw)) {
            return
          }
          const float = p.node.kind === 'float' || raw.includes('.')
          p.onChange(
            p.path,
            float
              ? { ...p.node, kind: 'float', value: raw }
              : { ...p.node, kind: 'int', value: raw },
          )
        }}
      />
      <Help node={p.node} />
    </Box>
  )
}

function TextValueField(p: FieldProps) {
  const node = p.node
  if (node.kind !== 'string' && node.kind !== 'keybind') {
    return null
  }
  return (
    <Box>
      <TextField
        size="small"
        fullWidth={true}
        label={p.label}
        value={node.value}
        error={node.kind === 'keybind' && node.value !== '' && !isKeybind(node.value)}
        onChange={(e) => p.onChange(p.path, { ...node, value: e.target.value })}
      />
      <Help node={p.node} />
    </Box>
  )
}

function ChoiceField(p: FieldProps) {
  const { t } = useLingui()
  const node = p.node
  if (node.kind !== 'choice') {
    return null
  }
  const selected = node.multiple
    ? node.value
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
    : node.value
  return (
    <Box>
      <Select
        size="small"
        fullWidth={true}
        displayEmpty={true}
        multiple={node.multiple}
        value={selected}
        label={p.label}
        onChange={(e) => {
          const next = e.target.value
          p.onChange(p.path, {
            ...node,
            value: Array.isArray(next) ? next.join(', ') : String(next),
          })
        }}
      >
        {node.allowBlank && !node.multiple ? <MenuItem value="">{t`Default`}</MenuItem> : null}
        {node.options.map((opt) => (
          <MenuItem key={opt} value={opt}>
            {opt}
          </MenuItem>
        ))}
      </Select>
      <Help node={p.node} />
      <Reset {...p} />
    </Box>
  )
}

function ListField(p: FieldProps) {
  const { t } = useLingui()
  if (p.node.kind !== 'list') {
    return null
  }
  const { items } = p.node
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Typography sx={text}>{p.label}</Typography>
      {items.map((item, i) => (
        <Box key={String(i)} sx={{ display: 'flex', alignItems: 'flex-start', gap: 0.5 }}>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Field
              label={String(i + 1)}
              node={item}
              path={[...p.path, i]}
              onChange={p.onChange}
              onOpen={p.onOpen}
            />
          </Box>
          <IconButton
            size="small"
            disabled={i === 0}
            onClick={() => {
              const next = [...items]
              const cur = next[i]
              const other = next[i - 1]
              if (cur && other) {
                next[i] = other
                next[i - 1] = cur
                p.onChange(p.path, { kind: 'list', items: next })
              }
            }}
          >
            <ChevronUp size={14} />
          </IconButton>
          <IconButton
            size="small"
            disabled={i === items.length - 1}
            onClick={() => {
              const next = [...items]
              const cur = next[i]
              const other = next[i + 1]
              if (cur && other) {
                next[i] = other
                next[i + 1] = cur
                p.onChange(p.path, { kind: 'list', items: next })
              }
            }}
          >
            <ChevronDown size={14} />
          </IconButton>
          <IconButton
            size="small"
            onClick={() =>
              p.onChange(p.path, { kind: 'list', items: items.filter((_, j) => j !== i) })
            }
          >
            <Trash2 size={14} />
          </IconButton>
        </Box>
      ))}
      <Button
        size="small"
        startIcon={<Plus size={14} />}
        onClick={() => p.onChange(p.path, { kind: 'list', items: [...items, emptyItem(items)] })}
        sx={noWrap}
      >
        {t`Add`}
      </Button>
    </Box>
  )
}

function Field(p: FieldProps) {
  switch (p.node.kind) {
    case 'bool':
      return <BoolField {...p} />
    case 'int':
    case 'float':
      return <NumberField {...p} />
    case 'string':
    case 'keybind':
      return <TextValueField {...p} />
    case 'choice':
      return <ChoiceField {...p} />
    case 'list':
      return <ListField {...p} />
    case 'object':
      return (
        <Accordion
          disableGutters={true}
          elevation={0}
          sx={{ bgcolor: 'transparent', '&:before': { display: 'none' } }}
        >
          <AccordionSummary expandIcon={<ChevronDown size={16} />}>
            <Typography sx={text}>{p.label}</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ display: 'flex', flexDirection: 'column', gap: 1, pl: 1 }}>
            <Fields tree={p.node} path={p.path} onChange={p.onChange} onOpen={p.onOpen} />
          </AccordionDetails>
        </Accordion>
      )
    case 'readonly':
      return (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          <Typography sx={text}>{p.label}</Typography>
          <TextField
            size="small"
            fullWidth={true}
            multiline={true}
            value={p.node.json}
            onChange={(e) => p.onChange(p.path, { kind: 'readonly', json: e.target.value })}
          />
          <ReadonlyLabel onOpen={p.onOpen} />
        </Box>
      )
    default: {
      const exhaustive: never = p.node
      return exhaustive
    }
  }
}

function ReadonlyLabel({ onOpen }: { onOpen: () => void }) {
  const { t } = useLingui()
  return (
    <Button size="small" variant="outlined" onClick={onOpen} sx={noWrap}>
      {t`Edit as JSON`}
    </Button>
  )
}

function Fields({
  tree,
  path,
  onChange,
  onOpen,
}: {
  tree: ConfigNode | null
  path?: readonly number[]
  onChange: (path: readonly number[], next: ConfigNode) => void
  onOpen: () => void
}) {
  const { t } = useLingui()
  const prefix = path ?? []
  if (tree?.kind === 'object') {
    const seen = new Set<string>()
    return tree.entries.map((entry, i) => {
      const section = sectionOf(entry.node)
      let header: ReactNode = null
      if (section && !seen.has(section)) {
        seen.add(section)
        header = (
          <Typography key={`s-${section}`} sx={{ ...text, fontWeight: 600, mt: 1 }}>
            {section}
          </Typography>
        )
      }
      return (
        <Box key={entry.key}>
          {header}
          <Field
            label={fieldLabel(entry.key)}
            node={entry.node}
            path={[...prefix, i]}
            onChange={onChange}
            onOpen={onOpen}
          />
        </Box>
      )
    })
  }
  if (tree) {
    return (
      <Field label={t`config.json`} node={tree} path={prefix} onChange={onChange} onOpen={onOpen} />
    )
  }
  return null
}

export { Fields }
