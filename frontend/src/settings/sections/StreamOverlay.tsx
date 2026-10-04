import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, InputAdornment, Switch, TextField, Tooltip } from '@mui/material'
import { Copy, Eye, EyeOff } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import {
  OverlayURL,
  RegenerateOverlayToken,
  SetOverlayEnabled,
  SetOverlayPort,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
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
    (err: unknown) => reportError(failCopy)(err),
  )
}

const WORLD_FIELDS: ReadonlySet<string> = new Set([
  'location',
  'season',
  'day',
  'year',
  'time',
  'date',
  'weather',
])

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
      .catch(reportError(failCopy))
  }
  type Field = (typeof OVERLAY_FIELDS)[number]
  const groups: { title: string; fields: Field[] }[] = [
    { title: t`World`, fields: OVERLAY_FIELDS.filter((f) => WORLD_FIELDS.has(f)) },
    {
      title: t`Player`,
      fields: OVERLAY_FIELDS.filter((f) => !(WORLD_FIELDS.has(f) || f.startsWith('skill.'))),
    },
    { title: t`Skills`, fields: OVERLAY_FIELDS.filter((f) => f.startsWith('skill.')) },
  ]
  const overall = overlayPreview(snapshot, undefined).kind
  const status = overall === 'value' ? '' : previewLine(overall, '', idle, wait)
  const row = (key: string, label: ReactNode, field?: Field) => {
    const preview = overlayPreview(snapshot, field)
    return (
      <Box key={key} sx={{ display: 'flex', alignItems: 'center', gap: 1, minHeight: 32 }}>
        <Box sx={{ width: 140, flexShrink: 0, fontSize: 13 }}>{label}</Box>
        <Box
          sx={{
            flexGrow: 1,
            minWidth: 0,
            fontSize: 13,
            color: 'var(--mortar-ink-sec)',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {preview.kind === 'value' ? preview.text : '\u2014'}
        </Box>
        <Tooltip title={copiedKey === key ? copied : t`Copy OBS URL`} placement="top">
          <span>
            <IconButton
              size="small"
              aria-label={t`Copy OBS URL`}
              disabled={!token}
              onClick={() => copyRow(field, key)}
            >
              <Copy size={16} />
            </IconButton>
          </span>
        </Tooltip>
      </Box>
    )
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1 }}>
        <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Values`}</Box>
        {status ? <Box sx={{ fontSize: 12, color: 'text.secondary' }}>{status}</Box> : null}
      </Box>
      {row('all', t`All values`)}
      {groups.map((g) => (
        <Box key={g.title} sx={{ display: 'flex', flexDirection: 'column' }}>
          <Box sx={{ fontSize: 12, fontWeight: 600, color: 'text.secondary', mt: 1 }}>
            {g.title}
          </Box>
          {g.fields.map((f) => row(f, <OverlayFieldLabel field={f} />, f))}
        </Box>
      ))}
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
    <Box sx={{ fontSize: 13, color: 'var(--mortar-ink-sec)' }}>
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
    <ConfirmDialog
      open={open}
      title={t`Regenerate token?`}
      body={t`Existing OBS sources stop working until their URLs are copied again.`}
      confirmLabel={t`Regenerate token`}
      color="error"
      onCancel={onClose}
      onConfirm={onConfirm}
    />
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
      <SettingRow label={t`Port`} description={t`The local port OBS reads the overlay from`}>
        <TextField
          type="number"
          size="small"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commitPort}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && e.target instanceof HTMLInputElement) {
              e.target.blur()
            }
          }}
          slotProps={{
            htmlInput: { min: MIN_PORT, max: MAX_PORT, step: 1, 'aria-label': t`Port` },
          }}
          sx={{ width: 120 }}
        />
      </SettingRow>
      <SettingRow
        label={t`Token`}
        description={t`Part of the OBS address; regenerate it if it leaks`}
      >
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
          <TextField
            type={shown ? 'text' : 'password'}
            size="small"
            value={token}
            slotProps={{
              htmlInput: { readOnly: true, 'aria-label': t`Token` },
              input: {
                endAdornment: (
                  <InputAdornment position="end">
                    <Tooltip title={shown ? t`Hide token` : t`Show token`}>
                      <IconButton
                        size="small"
                        aria-label={shown ? t`Hide token` : t`Show token`}
                        onClick={() => setShown((v) => !v)}
                        edge="end"
                      >
                        {shown ? <EyeOff size={16} /> : <Eye size={16} />}
                      </IconButton>
                    </Tooltip>
                    <Tooltip title={t`Copy token`}>
                      <span>
                        <IconButton
                          size="small"
                          aria-label={t`Copy token`}
                          disabled={!token}
                          onClick={() => copyText(token, push, copied, failCopy)}
                          edge="end"
                        >
                          <Copy size={16} />
                        </IconButton>
                      </span>
                    </Tooltip>
                  </InputAdornment>
                ),
              },
            }}
            sx={{ width: 240 }}
          />
          <Button variant="outlined" onClick={() => setConfirm(true)} sx={{ whiteSpace: 'nowrap' }}>
            {t`Regenerate…`}
          </Button>
        </Box>
      </SettingRow>
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
    <SettingsSection title={t`Stream overlay`}>
      <SettingRow
        label={t`Stream overlay`}
        description={t`Show live game info in OBS. Changes apply the next time you press Play.`}
      >
        <Switch
          checked={enabled}
          slotProps={{ input: { 'aria-label': t`Stream overlay` } }}
          onChange={(_, on) => persist(() => SetOverlayEnabled(on), push, fail)}
        />
      </SettingRow>
      {enabled ? (
        <SettingRow label={t`Show labels`}>
          <Switch
            checked={labels}
            slotProps={{ input: { 'aria-label': t`Show labels` } }}
            onChange={(_, on) => setLabels(on)}
          />
        </SettingRow>
      ) : null}
      {enabled ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, p: 2 }}>
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
        </Box>
      ) : null}
      {enabled ? <OverlayConnection port={port} token={token} push={push} fail={fail} /> : null}
    </SettingsSection>
  )
}
