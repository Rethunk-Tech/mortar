import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Typography,
} from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Users } from 'lucide-react'
import { useState } from 'react'
import type { FarmRow } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/models.ts'
import {
  CheckFarm,
  ExportFarm,
  FixFarm,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { listNames } from '../i18n/list.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

function FarmDialog({
  open,
  profile,
  onClose,
}: {
  open: boolean
  profile: Profile
  onClose: () => void
}) {
  const { t } = useLingui()
  const [text, setText] = useState('')
  const [host, setHost] = useState('')
  const [rows, setRows] = useState<FarmRow[] | null>(null)
  const [busy, setBusy] = useState(false)
  const { push } = useToasts.getState()
  const copyList = () =>
    ExportFarm('stardew', profile.id)
      .then(async (list) => {
        await Clipboard.SetText(JSON.stringify(list))
        push({
          kind: 'success',
          title: t`List copied`,
          body: t`Send it to the people joining you.`,
        })
      })
      .catch((error: unknown) => toastError(t`Could not make the list`, error))
  const check = () => {
    setBusy(true)
    CheckFarm('stardew', profile.id, text)
      .then((result) => {
        setHost(result.host)
        setRows(result.rows ?? [])
      })
      .catch((error: unknown) => toastError(t`Could not read that list`, error))
      .finally(() => setBusy(false))
  }
  const fix = (only: FarmRow | null) => {
    setBusy(true)
    FixFarm('stardew', profile.id, text, only ? [only.id] : null)
      .then((result) => {
        const manual = result.manual ?? []
        const count = result.queued
        push({
          kind: manual.length > 0 ? 'warning' : 'success',
          title: t`${count} to download`,
          ...(manual.length > 0 ? { body: t`Install by hand: ${listNames(manual)}` } : {}),
        })
      })
      .catch((error: unknown) => toastError(t`Could not queue the downloads`, error))
      .finally(() => setBusy(false))
  }
  const done = () => {
    setText('')
    setRows(null)
    onClose()
  }
  const detail = (row: FarmRow) => {
    if (row.state === 'off') {
      return t`Disabled in this profile`
    }
    return row.state === 'different' ? t`Host ${row.host}, yours ${row.mine}` : t`Host ${row.host}`
  }
  const action = (row: FarmRow) => (row.state === 'different' ? t`Update` : t`Install`)
  return (
    <Dialog open={open} onClose={done} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Multiplayer mod list`}</DialogTitle>
      <DialogContent dividers={true}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          <Button disabled={busy} sx={{ alignSelf: 'flex-start' }} onClick={copyList}>
            {t`Copy this profile's list`}
          </Button>
          <TextField
            fullWidth={true}
            multiline={true}
            minRows={3}
            maxRows={6}
            size="small"
            label={t`The host's list`}
            value={text}
            onChange={(event) => {
              setText(event.target.value)
              setRows(null)
            }}
          />
          {rows?.length === 0 ? <Typography>{t`This profile matches ${host}.`}</Typography> : null}
          {rows?.map((row) => (
            <Box key={row.id} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Typography sx={{ flex: 1, fontSize: 14 }}>
                {row.name}
                <Typography component="span" color="text.secondary" sx={{ fontSize: 13, ml: 1 }}>
                  {detail(row)}
                </Typography>
              </Typography>
              {row.fixable ? (
                <Button size="small" disabled={busy} onClick={() => fix(row)}>
                  {action(row)}
                </Button>
              ) : null}
            </Box>
          ))}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={done}>{t`Close`}</Button>
        {rows?.some((row) => row.fixable) ? (
          <Button disabled={busy} onClick={() => fix(null)}>
            {t`Fix all`}
          </Button>
        ) : null}
        <Button variant="contained" disabled={busy || text.trim() === ''} onClick={check}>
          {t`Check`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

// Stardew multiplayer: copy the mods a guest must match, or check a host's list against this profile and fetch what
// differs.
export function FarmMenuItem({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const [open, setOpen] = useState(false)
  if (game?.id !== 'stardew') {
    return null
  }
  return (
    <>
      <MenuAction
        icon={<Users size={16} />}
        label={t`Multiplayer mod list…`}
        onClick={() => {
          close()
          setOpen(true)
        }}
      />
      <FarmDialog open={open} profile={profile} onClose={() => setOpen(false)} />
    </>
  )
}
