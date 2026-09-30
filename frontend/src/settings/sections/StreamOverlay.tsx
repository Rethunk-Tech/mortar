import { useLingui } from '@lingui/react/macro'
import { Box, Button, FormControlLabel, Switch, TextField } from '@mui/material'
import { Copy, RefreshCw } from 'lucide-react'
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

const MIN_PORT = 1024
const MAX_PORT = 65_535
const button = { whiteSpace: 'nowrap', alignSelf: 'flex-start' }

function persist(
  run: () => Promise<void>,
  push: ReturnType<typeof useToasts.getState>['push'],
  title: string,
) {
  run().catch((err: unknown) => {
    const body = errorText(err)
    push({ kind: 'error', title, ...(body ? { body } : {}) })
  })
}

export function StreamOverlay() {
  const { t } = useLingui()
  const enabled = useSettings((s) => s.overlayEnabled)
  const port = useSettings((s) => s.overlayPort)
  const token = useSettings((s) => s.overlayToken)
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  const [draft, setDraft] = useState(String(port))
  const [shown, setShown] = useState(false)
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
  const copy = (text: string, title: string) => {
    navigator.clipboard.writeText(text).then(
      () => push({ kind: 'success', title }),
      (err: unknown) => {
        const body = errorText(err)
        push({ kind: 'error', title: t`Could not copy`, ...(body ? { body } : {}) })
      },
    )
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <FormControlLabel
        sx={{ m: 0, alignItems: 'center' }}
        control={
          <Switch
            checked={enabled}
            onChange={(_, on) => persist(() => SetOverlayEnabled(on), push, fail)}
          />
        }
        label={t`Enable`}
      />
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
        slotProps={{ htmlInput: { readOnly: true } }}
        sx={{ alignSelf: 'flex-start', width: 320 }}
      />
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
        <Button variant="outlined" onClick={() => setShown((v) => !v)} sx={button}>
          {shown ? t`Hide` : t`Show`}
        </Button>
        <Button
          variant="outlined"
          startIcon={<Copy size={16} />}
          disabled={!token}
          onClick={() => copy(token, t`Token copied`)}
          sx={button}
        >
          {t`Copy token`}
        </Button>
        <Button
          variant="outlined"
          startIcon={<Copy size={16} />}
          disabled={!(enabled && token)}
          onClick={() => {
            OverlayURL()
              .then((url) => copy(url, t`Overlay URL copied`))
              .catch((err: unknown) => {
                const body = errorText(err)
                push({ kind: 'error', title: t`Could not copy`, ...(body ? { body } : {}) })
              })
          }}
          sx={button}
        >
          {t`Copy overlay URL`}
        </Button>
        <Button
          variant="outlined"
          startIcon={<RefreshCw size={16} />}
          onClick={() => persist(() => RegenerateOverlayToken(), push, fail)}
          sx={button}
        >
          {t`Regenerate`}
        </Button>
      </Box>
      <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
        {t`The change applies at the next launch.`}
      </Box>
    </Box>
  )
}
