import { useLingui } from '@lingui/react/macro'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  InputAdornment,
  Switch,
  TextField,
  Tooltip,
} from '@mui/material'
import { ChevronDown, Copy, Eye, EyeOff } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  OverlayURL,
  RegenerateOverlayToken,
  SetOverlayEnabled,
  SetOverlayPort,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { errorText } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { useSettings } from '../store.ts'
import {
  OVERLAY_EXAMPLE_CSS,
  OVERLAY_FIELDS,
  type OverlayPreview,
  overlayPageUrl,
  overlayPreview,
} from './overlayUrl.ts'

const MIN_PORT = 1024
const MAX_PORT = 65_535
const POLL_MS = 2000
const COPIED_MS = 1500

type Push = ReturnType<typeof useToasts.getState>['push']
type Snapshot = { ok: true; body: Record<string, unknown> } | { ok: false } | null

function persist(run: () => Promise<void>, push: Push, title: string) {
  run().catch((err: unknown) => {
    const body = errorText(err)
    push({ kind: 'error', title, ...(body ? { body } : {}) })
  })
}

function previewLine(kind: OverlayPreview['kind'], value: string, idle: string, wait: string) {
  if (kind === 'unreachable') {
    return wait
  }
  if (kind === 'notInGame') {
    return idle
  }
  return value
}

function copyText(text: string, push: Push, copied: string, failCopy: string) {
  return navigator.clipboard.writeText(text).then(
    () => push({ kind: 'success', title: copied }),
    (err: unknown) => {
      const body = errorText(err)
      push({ kind: 'error', title: failCopy, ...(body ? { body } : {}) })
    },
  )
}

function OverlayFieldLabel({ field }: { field: (typeof OVERLAY_FIELDS)[number] }) {
  const { t } = useLingui()
  switch (field) {
    case 'location':
      return t`Location`
    case 'player':
      return t`Player`
    case 'season':
      return t`Season`
    case 'day':
      return t`Day`
    case 'year':
      return t`Year`
    case 'time':
      return t`Time`
    case 'date':
      return t`Date`
    case 'money':
      return t`Money`
    case 'weather':
      return t`Weather`
    case 'health':
      return t`Health`
    case 'stamina':
      return t`Stamina`
    case 'skill.farming':
      return t`Farming skill`
    case 'skill.fishing':
      return t`Fishing skill`
    case 'skill.foraging':
      return t`Foraging skill`
    case 'skill.mining':
      return t`Mining skill`
    case 'skill.combat':
      return t`Combat skill`
    case 'skill.luck':
      return t`Luck skill`
    default:
      return field
  }
}

function OverlayValues({
  snapshot,
  labels,
  token,
  idle,
  wait,
  copied,
  failCopy,
  push,
}: {
  snapshot: Snapshot
  labels: boolean
  token: string
  idle: string
  wait: string
  copied: string
  failCopy: string
  push: Push
}) {
  const { t } = useLingui()
  const [copiedKey, setCopiedKey] = useState('')
  const copy = (text: string, key: string) => {
    copyText(text, push, copied, failCopy).then(() => {
      setCopiedKey(key)
      globalThis.setTimeout(() => setCopiedKey((cur) => (cur === key ? '' : cur)), COPIED_MS)
    })
  }
  const copyRow = (field: string | undefined, key: string) => {
    OverlayURL()
      .then((base) => copy(overlayPageUrl(base, field, labels), key))
      .catch((err: unknown) => {
        const body = errorText(err)
        push({ kind: 'error', title: failCopy, ...(body ? { body } : {}) })
      })
  }
  const rows: { key: string; field?: (typeof OVERLAY_FIELDS)[number] }[] = [
    { key: 'all' },
    ...OVERLAY_FIELDS.map((field) => ({ key: field, field })),
  ]
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Values`}</Box>
      {rows.map((row) => {
        const preview = overlayPreview(snapshot, row.field)
        const line = previewLine(
          preview.kind,
          preview.kind === 'value' ? preview.text : '',
          idle,
          wait,
        )
        return (
          <Box key={row.key} sx={{ display: 'flex', alignItems: 'center', gap: 1, minHeight: 32 }}>
            <Box sx={{ width: 140, flexShrink: 0, fontSize: 13 }}>
              {row.field ? <OverlayFieldLabel field={row.field} /> : t`All values`}
            </Box>
            <Box
              sx={{
                flexGrow: 1,
                minWidth: 0,
                fontSize: 13,
                color: 'rgba(225,225,230,0.95)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {line}
            </Box>
            <Tooltip title={copiedKey === row.key ? copied : t`Copy OBS URL`} placement="top">
              <IconButton
                size="small"
                aria-label={t`Copy OBS URL`}
                disabled={!token}
                onClick={() => copyRow(row.field, row.key)}
              >
                <Copy size={16} />
              </IconButton>
            </Tooltip>
          </Box>
        )
      })}
    </Box>
  )
}

function OverlayHowTo({
  push,
  copied,
  failCopy,
}: {
  push: Push
  copied: string
  failCopy: string
}) {
  const { t } = useLingui()
  const [done, setDone] = useState(false)
  return (
    <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
      {t`In OBS: Sources, +, Browser, paste the URL. Width 400, height 80. Style with Custom CSS.`}
      <Button
        variant="text"
        size="small"
        startIcon={<Copy size={14} />}
        onClick={() => {
          copyText(OVERLAY_EXAMPLE_CSS, push, copied, failCopy).then(() => {
            setDone(true)
            globalThis.setTimeout(() => setDone(false), COPIED_MS)
          })
        }}
        sx={{ textTransform: 'none', ml: 1 }}
      >
        {done ? copied : t`Copy CSS`}
      </Button>
    </Box>
  )
}

function OverlayRegenDialog({
  open,
  onClose,
  onConfirm,
}: {
  open: boolean
  onClose: () => void
  onConfirm: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>{t`Regenerate token?`}</DialogTitle>
      <DialogContent>
        {t`Existing OBS sources stop working until their URLs are copied again.`}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} sx={{ textTransform: 'none' }}>
          {t`Cancel`}
        </Button>
        <Button onClick={onConfirm} sx={{ textTransform: 'none' }}>
          {t`Regenerate token`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function OverlayConnection({
  port,
  token,
  push,
  fail,
}: {
  port: number
  token: string
  push: Push
  fail: string
}) {
  const { t } = useLingui()
  const copied = t`Copied`
  const failCopy = t`Could not copy`
  const [draft, setDraft] = useState(String(port))
  const [shown, setShown] = useState(false)
  const [confirm, setConfirm] = useState(false)
  useEffect(() => setDraft(String(port)), [port])
  const commitPort = () => {
    const n = Number(draft)
    if (!Number.isInteger(n) || n < MIN_PORT || n > MAX_PORT) {
      setDraft(String(port))
      return
    }
    if (n !== port) {
      persist(() => SetOverlayPort(n), push, fail)
    }
  }
  return (
    <>
      <Accordion
        disableGutters={true}
        sx={{ bgcolor: 'transparent', boxShadow: 'none', '&:before': { display: 'none' } }}
      >
        <AccordionSummary
          expandIcon={<ChevronDown size={16} />}
          sx={{ px: 0, minHeight: 36, '& .MuiAccordionSummary-content': { my: 0.5 } }}
        >
          <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Connection`}</Box>
        </AccordionSummary>
        <AccordionDetails sx={{ px: 0, pt: 0, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          <TextField
            type="number"
            size="small"
            label={t`Port`}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onBlur={commitPort}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && e.target instanceof HTMLInputElement) {
                e.target.blur()
              }
            }}
            slotProps={{ htmlInput: { min: MIN_PORT, max: MAX_PORT, step: 1 } }}
            sx={{ alignSelf: 'flex-start', width: 320 }}
          />
          <TextField
            type={shown ? 'text' : 'password'}
            size="small"
            label={t`Token`}
            value={token}
            slotProps={{
              htmlInput: { readOnly: true },
              input: {
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      size="small"
                      aria-label={shown ? t`Hide token` : t`Show token`}
                      onClick={() => setShown((v) => !v)}
                      edge="end"
                    >
                      {shown ? <EyeOff size={16} /> : <Eye size={16} />}
                    </IconButton>
                    <IconButton
                      size="small"
                      aria-label={t`Copy token`}
                      disabled={!token}
                      onClick={() => copyText(token, push, copied, failCopy)}
                      edge="end"
                    >
                      <Copy size={16} />
                    </IconButton>
                  </InputAdornment>
                ),
              },
            }}
            sx={{ alignSelf: 'flex-start', width: 320 }}
          />
          <Button
            variant="text"
            onClick={() => setConfirm(true)}
            sx={{ textTransform: 'none', alignSelf: 'flex-start' }}
          >
            {t`Regenerate token`}
          </Button>
        </AccordionDetails>
      </Accordion>
      <OverlayRegenDialog
        open={confirm}
        onClose={() => setConfirm(false)}
        onConfirm={() => {
          setConfirm(false)
          persist(() => RegenerateOverlayToken(), push, fail)
        }}
      />
    </>
  )
}

function useOverlaySnapshot(enabled: boolean, port: number, token: string): Snapshot {
  const [snapshot, setSnapshot] = useState<Snapshot>(null)
  useEffect(() => {
    if (!enabled) {
      setSnapshot(null)
      return
    }
    let live = true
    const tick = () => {
      const url = `http://127.0.0.1:${port}/state?token=${encodeURIComponent(token)}`
      fetch(url)
        .then(async (res) => {
          if (!res.ok) {
            throw new Error('bad')
          }
          return (await res.json()) as Record<string, unknown>
        })
        .then((body) => {
          if (live) {
            setSnapshot({ ok: true, body })
          }
        })
        .catch(() => {
          if (live) {
            setSnapshot({ ok: false })
          }
        })
    }
    tick()
    const id = setInterval(tick, POLL_MS)
    return () => {
      live = false
      clearInterval(id)
    }
  }, [enabled, port, token])
  return snapshot
}

export function StreamOverlay() {
  const { t } = useLingui()
  const enabled = useSettings((s) => s.overlayEnabled)
  const port = useSettings((s) => s.overlayPort)
  const token = useSettings((s) => s.overlayToken)
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  const idle = t`Load a save to see values`
  const wait = t`Start the game to see values`
  const copied = t`Copied`
  const failCopy = t`Could not copy`
  const [labels, setLabels] = useState(false)
  const snapshot = useOverlaySnapshot(enabled, port, token)
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1.5,
        p: '14px',
        bgcolor: 'rgba(55,55,65,0.9)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <Box
          sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', gap: '2px', minWidth: 0 }}
        >
          <Box sx={{ fontSize: 15, fontWeight: 600 }}>{t`Stream overlay`}</Box>
          <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
            {t`Show live game info in OBS. Changes apply the next time you press Play.`}
          </Box>
        </Box>
        <FormControlLabel
          sx={{ m: 0, flexShrink: 0 }}
          control={
            <Switch
              checked={enabled}
              onChange={(_, on) => persist(() => SetOverlayEnabled(on), push, fail)}
            />
          }
          label={t`Enable`}
        />
      </Box>
      {enabled ? (
        <>
          <FormControlLabel
            sx={{ m: 0 }}
            control={<Switch checked={labels} onChange={(_, on) => setLabels(on)} />}
            label={t`Show labels`}
          />
          <OverlayValues
            snapshot={snapshot}
            labels={labels}
            token={token}
            idle={idle}
            wait={wait}
            copied={copied}
            failCopy={failCopy}
            push={push}
          />
          <OverlayHowTo push={push} copied={copied} failCopy={failCopy} />
        </>
      ) : null}
      <OverlayConnection port={port} token={token} push={push} fail={fail} />
    </Box>
  )
}
