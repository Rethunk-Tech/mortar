import { useLingui } from '@lingui/react/macro'
import { Box, Button, InputAdornment, TextField } from '@mui/material'
import { Copy, Eye, EyeOff } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import {
  OverlayURL,
  RegenerateOverlayToken,
  SetOverlayEnabled,
  SetOverlayPort,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { copyText } from '../../share/copyText.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { TipIconButton } from '../../shell/TipIconButton.tsx'
import { space } from '../../theme/density.ts'
import { reportError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSwitch } from '../PrefControls.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import {
  OVERLAY_EXAMPLE_CSS,
  type OverlayPreview,
  overlayFields,
  overlayPageUrl,
  overlayPreview,
  STARDEW_FIELDS,
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

const WORLD_FIELDS: ReadonlySet<string> = new Set([
  'location',
  'season',
  'day',
  'year',
  'time',
  'date',
  'weather',
])

function OverlayFieldLabel({ field }: { field: string }) {
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
    case 'moon':
      return t`Moon`
    case 'crew':
      return t`Crew`
    case 'quota':
      return t`Quota`
    case 'daysLeft':
      return t`Days left`
    case 'credits':
      return t`Credits`
    case 'biome':
      return t`Biome`
    case 'bosses':
      return t`Bosses defeated`
    case 'bossList':
      return t`Boss names`
    default:
      return field
  }
}

function OverlayValues({
  game,
  snapshot,
  labels,
  token,
  idle,
  wait,
  copied,
  failCopy,
}: {
  game: string
  snapshot: Snapshot
  labels: boolean
  token: string
  idle: string
  wait: string
  copied: string
  failCopy: string
}) {
  const { t } = useLingui()
  const [copiedKey, setCopiedKey] = useState('')
  const copy = (text: string, key: string) => {
    copyText(text, copied, () => {
      setCopiedKey(key)
      globalThis.setTimeout(() => setCopiedKey((cur) => (cur === key ? '' : cur)), COPIED_MS)
    })
  }
  const copyRow = (field: string | undefined, key: string) => {
    OverlayURL()
      .then((base) => copy(overlayPageUrl(base, field, labels), key))
      .catch(reportError(failCopy))
  }
  type Field = string
  const fields = overlayFields(game)
  const groups: { title: string; fields: Field[] }[] =
    fields === STARDEW_FIELDS
      ? [
          { title: t`World`, fields: fields.filter((f) => WORLD_FIELDS.has(f)) },
          {
            title: t`Player`,
            fields: fields.filter((f) => !(WORLD_FIELDS.has(f) || f.startsWith('skill.'))),
          },
          { title: t`Skills`, fields: fields.filter((f) => f.startsWith('skill.')) },
        ]
      : [
          { title: t`Player`, fields: fields.filter((f) => f === 'player') },
          { title: t`Game`, fields: fields.filter((f) => f !== 'player') },
        ]
  const overall = overlayPreview(snapshot, undefined).kind
  const status = overall === 'value' ? '' : previewLine(overall, '', idle, wait)
  const row = (key: string, label: ReactNode, field?: Field) => {
    const preview = overlayPreview(snapshot, field)
    return (
      <Box
        key={key}
        sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minHeight: space.control }}
      >
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
        <TipIconButton
          label={copiedKey === key ? copied : t`Copy OBS URL`}
          disabled={!token}
          onClick={() => copyRow(field, key)}
        >
          <Copy size={16} />
        </TipIconButton>
      </Box>
    )
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'baseline', gap: space.gap }}>
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

function OverlayHowTo({ copied }: { copied: string }) {
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
          copyText(OVERLAY_EXAMPLE_CSS, copied, () => {
            setDone(true)
            globalThis.setTimeout(() => setDone(false), COPIED_MS)
          })
        }}
        sx={{ ml: 1 }}
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
  const [draft, setDraft] = useState(String(port))
  const [portError, setPortError] = useState(false)
  const [shown, setShown] = useState(false)
  const [confirm, setConfirm] = useState(false)
  useEffect(() => {
    setDraft(String(port))
    setPortError(false)
  }, [port])
  const portRange = t`Enter a number from ${{ min: MIN_PORT }} to ${{ max: MAX_PORT }}`
  const commitPort = () => {
    const n = Number(draft)
    if (!Number.isInteger(n) || n < MIN_PORT || n > MAX_PORT) {
      setPortError(true)
      return
    }
    setPortError(false)
    if (n !== port) {
      persist(() => SetOverlayPort(n), fail)
    }
  }
  return (
    <>
      <SettingRow label={t`Port`} description={t`The local port OBS reads the overlay from`}>
        <TextField
          type="number"
          size="small"
          value={draft}
          error={portError}
          helperText={portError ? portRange : undefined}
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
        <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'center' }}>
          <TextField
            type={shown ? 'text' : 'password'}
            size="small"
            value={token}
            slotProps={{
              htmlInput: { readOnly: true, 'aria-label': t`Token` },
              input: {
                endAdornment: (
                  <InputAdornment position="end">
                    <TipIconButton
                      label={shown ? t`Hide token` : t`Show token`}
                      onClick={() => setShown((v) => !v)}
                      edge="end"
                    >
                      {shown ? <EyeOff size={16} /> : <Eye size={16} />}
                    </TipIconButton>
                    <TipIconButton
                      label={t`Copy token`}
                      disabled={!token}
                      onClick={() => copyText(token, copied)}
                      edge="end"
                    >
                      <Copy size={16} />
                    </TipIconButton>
                  </InputAdornment>
                ),
              },
            }}
            sx={{ width: 240 }}
          />
          <Button variant="outlined" onClick={() => setConfirm(true)}>
            {t`Regenerate…`}
          </Button>
        </Box>
      </SettingRow>
      <OverlayRegenDialog
        open={confirm}
        onClose={() => setConfirm(false)}
        onConfirm={() => {
          setConfirm(false)
          persist(
            () =>
              RegenerateOverlayToken().then(() => {
                push({ kind: 'success', title: t`Token regenerated` })
              }),
            fail,
          )
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

export function StreamOverlay({ game }: { game: string }) {
  const { t } = useLingui()
  const enabled = useSettings((s) => s.overlayEnabled)
  const port = useSettings((s) => s.overlayPort)
  const token = useSettings((s) => s.overlayToken)
  const push = useToasts((s) => s.push)
  const fail = t`Could not save that setting`
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
        <PrefSwitch
          checked={enabled}
          onChange={(on) => persist(() => SetOverlayEnabled(on), fail)}
          label={t`Stream overlay`}
        />
      </SettingRow>
      {enabled ? (
        <SettingRow label={t`Show labels`}>
          <PrefSwitch checked={labels} onChange={(on) => setLabels(on)} label={t`Show labels`} />
        </SettingRow>
      ) : null}
      {enabled ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, p: space.pad }}>
          <OverlayValues
            game={game}
            snapshot={snapshot}
            labels={labels}
            token={token}
            idle={idle}
            wait={wait}
            copied={copied}
            failCopy={failCopy}
          />
          <OverlayHowTo copied={copied} />
        </Box>
      ) : null}
      {enabled ? <OverlayConnection port={port} token={token} push={push} fail={fail} /> : null}
    </SettingsSection>
  )
}
