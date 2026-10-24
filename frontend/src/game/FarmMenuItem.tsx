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
import { ReadTextFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { listNames } from '../i18n/list.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { space } from '../theme/density.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { hasModList } from './modList.ts'
import { openProfileDialog } from './profileDialog.ts'

const section = { fontSize: 'base.fontSize', fontWeight: 700 } as const

export function FarmDialog({
  open,
  profile,
  onClose,
}: {
  open: boolean
  profile: Profile
  onClose: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [text, setText] = useState('')
  const [host, setHost] = useState('')
  const [rows, setRows] = useState<FarmRow[] | null>(null)
  const [busy, run] = usePending()
  const { push } = useToasts.getState()
  const copyList = () =>
    ExportFarm(game, profile.id)
      .then(async (list) => {
        await Clipboard.SetText(JSON.stringify(list))
        push({
          kind: 'success',
          title: t`List copied`,
          body: t`Send it to the people joining you.`,
        })
      })
      .catch((error: unknown) => toastError(t`Could not make the list`, error))
  const loadList = () =>
    ReadTextFile(t`Open the host's list`)
      .then((loaded) => {
        if (loaded) {
          setText(loaded)
          setRows(null)
        }
      })
      .catch((error: unknown) => toastError(t`Could not read that file`, error))
  const check = () =>
    run(
      () =>
        CheckFarm(game, profile.id, text).then((result) => {
          setHost(result.host)
          setRows(result.rows ?? [])
        }),
      { errorTitle: t`Could not read that list` },
    )
  const fix = (only: FarmRow | null) =>
    run(
      () =>
        FixFarm(game, profile.id, text, only ? [only.id] : null).then((result) => {
          const manual = result.manual ?? []
          const count = result.queued
          push({
            kind: manual.length > 0 ? 'warning' : 'success',
            title: t`${count} to download`,
            ...(manual.length > 0
              ? { body: t`Install by hand: ${{ names: listNames(manual) }}` }
              : {}),
          })
        }),
      { errorTitle: t`Could not queue the downloads` },
    )
  const done = () => {
    setText('')
    setRows(null)
    onClose()
  }
  const detail = (row: FarmRow) => {
    if (row.state === 'off') {
      return t`Disabled in this profile`
    }
    return row.state === 'different'
      ? t`Host ${{ host: row.host }}, yours ${{ mine: row.mine }}`
      : t`Host ${{ host: row.host }}`
  }
  // Once a check finds something to fix, fixing is what the dialog is for.
  const fixable = rows?.some((row) => row.fixable) === true
  const action = (row: FarmRow) => (row.state === 'different' ? t`Update` : t`Add`)
  return (
    <Dialog open={open} onClose={done} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Multiplayer mod list`}</DialogTitle>
      <DialogContent dividers={true}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
          <Typography sx={section}>{t`Hosting`}</Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, flexWrap: 'wrap' }}>
            <Button variant="outlined" disabled={busy} onClick={copyList}>
              {t`Copy this profile's list`}
            </Button>
            <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
              {t`Send it to the people joining you.`}
            </Typography>
          </Box>
          <Typography sx={{ ...section, mt: 1 }}>{t`Joining`}</Typography>
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
          <Button
            variant="outlined"
            disabled={busy}
            onClick={loadList}
            sx={{ alignSelf: 'flex-start' }}
          >
            {t`Load the host's list from a file…`}
          </Button>
          {rows?.length === 0 ? (
            <Typography>
              {host ? t`This profile matches ${host}.` : t`This profile matches that list.`}
            </Typography>
          ) : null}
          {rows?.map((row) => (
            <Box key={row.id} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
              <Typography sx={{ flex: 1, fontSize: 'medium.fontSize' }}>
                {row.name}
                <Typography
                  component="span"
                  color="text.secondary"
                  sx={{ fontSize: 'base.fontSize', ml: 1 }}
                >
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
        {fixable ? (
          <Button variant="contained" disabled={busy} onClick={() => fix(null)}>
            {t`Fix all`}
          </Button>
        ) : (
          <Button variant="contained" disabled={busy || text.trim() === ''} onClick={check}>
            {t`Check`}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  )
}

// Multiplayer: copy the mods a guest must match, or check a host's list against this profile and fetch what differs.
export function FarmMenuItem({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const listed = useProfiles((s) => hasModList(s.game ?? undefined))
  if (!listed) {
    return null
  }
  return (
    <MenuAction
      icon={<Users size={16} />}
      label={t`Multiplayer mod list…`}
      onClick={() => {
        close()
        openProfileDialog('farm', profile.id)
      }}
    />
  )
}
