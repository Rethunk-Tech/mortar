import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Copy } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  GrantSteamAccess,
  SteamAccess,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import { MONO } from '../theme/theme.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { ConfirmDialog } from './ConfirmDialog.tsx'

export function FlatpakGrant() {
  const { t } = useLingui()
  const [cmd, setCmd] = useState('')
  const [needed, setNeeded] = useState(false)
  const [granted, setGranted] = useState(false)
  const [ask, setAsk] = useState(false)
  const load = () => {
    SteamAccess()
      .then((a) => {
        setCmd(a.command)
        setNeeded(a.needed)
        setGranted(a.granted)
      })
      .catch(reportUnexpected)
  }
  useEffect(load, [])
  if (!needed || granted || !cmd) {
    return null
  }
  return (
    <>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, px: 2.5, py: 1.5 }}>
        <Box sx={{ fontSize: 16 }}>{t`Flatpak Steam cannot read your mods`}</Box>
        <Box sx={{ fontSize: 13, color: 'var(--mortar-ink-sec)' }}>
          {t`Grant the Steam sandbox read access to Mortar's data folder, or SMAPI will not see this profile's mods.`}
        </Box>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start' }}>
          <Box
            sx={{
              flex: 1,
              minWidth: 0,
              px: 1.5,
              py: 0.75,
              bgcolor: 'var(--mortar-overlay-45)',
              border: '1px solid var(--mortar-hairline-15)',
              borderRadius: '6px',
              fontFamily: MONO,
              fontSize: 13,
              wordBreak: 'break-all',
              userSelect: 'text',
            }}
          >
            {cmd}
          </Box>
          <Button
            variant="outlined"
            startIcon={<Copy size={16} />}
            onClick={() => {
              Clipboard.SetText(cmd).then(
                () => useToasts.getState().push({ kind: 'success', title: t`Command copied` }),
                reportUnexpected,
              )
            }}
            sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
          >
            {t`Copy`}
          </Button>
          <Button variant="contained" onClick={() => setAsk(true)} sx={{ whiteSpace: 'nowrap' }}>
            {t`Grant access`}
          </Button>
        </Box>
      </Box>
      <ConfirmDialog
        open={ask}
        title={t`Grant Flatpak Steam access?`}
        body={t`This runs the command below once for your user. Steam will then be able to read Mortar's data folder.`}
        confirmLabel={t`Grant access`}
        onCancel={() => setAsk(false)}
        onConfirm={() => {
          setAsk(false)
          GrantSteamAccess()
            .then(() => {
              useToasts.getState().push({ kind: 'success', title: t`Access granted` })
              load()
            })
            .catch(reportUnexpected)
        }}
      >
        <Box sx={{ mt: 1.5, fontFamily: MONO, fontSize: 13, userSelect: 'text' }}>{cmd}</Box>
      </ConfirmDialog>
    </>
  )
}
