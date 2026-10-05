import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
  ListItemIcon,
  Menu,
  MenuItem,
} from '@mui/material'
import { UserRound } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import type { Arrival } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { formatKb } from '../i18n/bytes.ts'
import { useNexusDetails } from '../mods/nexusDetails.ts'
import { useProfiles } from '../profiles/store.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { type InlineError, inlineError, reportUnexpected } from '../toasts/report.ts'
import { arrivalGame, arrivalName, fallbackName, useNxm } from './store.ts'

function useModName(arrival: Arrival): string {
  const [name, setName] = useState(() => arrival.package || fallbackName(arrival.link.modId))
  useEffect(() => {
    let live = true
    arrivalName(arrival)
      .then((n) => live && setName(n))
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [arrival])
  return name
}

function ArrivalPrompt({ arrival }: { arrival: Arrival }) {
  const { t } = useLingui()
  const openGameId = useProfiles((s) => s.game?.id)
  const game = arrivalGame(arrival, openGameId)
  const site = arrival.package ? 'Thunderstore' : 'Nexus'
  const choose = useNxm((s) => s.choose)
  const arrivals = useNxm((s) => s.arrivals)
  const dismiss = useNxm((s) => s.dismiss)
  const lastId = useSettings((s) => gamePrefs(s, game).nxmDefaultProfile || s.lastProfile?.[game])
  const name = useModName(arrival)
  const file = useNexusDetails((s) =>
    s.byId[arrival.link.modId]?.details?.files?.find((f) => f.fileId === arrival.link.fileId),
  )
  // New profile creates in the game the profiles store has open, so it is offered only when that is this game.
  const canCreate = useProfiles((s) => s.game?.id === game)
  const gameName = useProfiles((s) => (s.game?.id === game ? s.game.name : game))
  const [profiles, setProfiles] = useState<Profile[] | null>(null)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [creating, setCreating] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<InlineError | null>(null)
  const [applyAll, setApplyAll] = useState(false)
  const load = useCallback(() => {
    List(game)
      .then((list) => setProfiles((list ?? []).filter((p) => !p.hidden)))
      .catch((e: unknown) => {
        setProfiles([])
        reportUnexpected(e)
      })
  }, [game])
  useEffect(load, [load])
  const open = profiles?.find((p) => p.id === lastId) ?? profiles?.[0]
  const others = profiles?.filter((p) => p.id !== open?.id) ?? []
  const pick = (profile: string) => {
    if (busy) {
      return
    }
    setBusy(true)
    setError(null)
    const ids = applyAll ? arrivals.map((a) => a.id) : [arrival.id]
    Promise.all(ids.map((id) => choose(id, profile)))
      .catch((e: unknown) => setError(inlineError(e)))
      .finally(() => setBusy(false))
  }
  let text = t`You started this download on ${site}. Choose the profile it goes into.`
  if (profiles === null) {
    text = t`Loading profiles…`
  } else if (profiles.length === 0) {
    text = canCreate
      ? t`You started this download on ${site}. Create a profile to put it in first.`
      : t`You started this download on ${site}, but you have no ${gameName} profile yet.`
  }
  return (
    <Dialog
      open={true}
      onClose={(_, reason) => {
        if (reason === 'escapeKeyDown' && !busy) {
          dismiss(arrival.id)
        }
      }}
      slotProps={{ paper: { sx: { width: 460, maxWidth: 'calc(100% - 32px)' } } }}
    >
      <DialogTitle>{t`Install ${name}?`}</DialogTitle>
      <DialogContent>
        {file ? (
          <DialogContentText sx={{ mb: 1 }}>
            {t`${file.fileName} · ${formatKb(file.sizeKb)}`}
          </DialogContentText>
        ) : null}
        {arrivals.length > 1 ? (
          <DialogContentText sx={{ mb: 1 }}>
            {t`${arrivals.length - 1} more waiting`}
          </DialogContentText>
        ) : null}
        {profiles === null ? (
          <LoadingRow>{text}</LoadingRow>
        ) : (
          <DialogContentText>{text}</DialogContentText>
        )}
        {error ? (
          <DialogContentText color="error" title={error.details} sx={{ mt: 1 }}>
            {error.message}
          </DialogContentText>
        ) : null}
        {arrivals.length > 1 ? (
          <FormControlLabel
            control={
              <Checkbox checked={applyAll} onChange={(_, checked) => setApplyAll(checked)} />
            }
            label={t`Apply this profile to all waiting`}
          />
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button disabled={busy} onClick={() => dismiss(arrival.id)}>
          {t`Ignore`}
        </Button>
        {others.length > 0 ? (
          <Button
            disabled={busy}
            aria-haspopup="menu"
            aria-expanded={anchor !== null}
            onClick={(e) => setAnchor(e.currentTarget)}
          >
            {t`Other profile…`}
          </Button>
        ) : null}
        {open ? (
          <Button variant="contained" disabled={busy} onClick={() => pick(open.id)}>
            {t`Install into ${open.name}`}
          </Button>
        ) : null}
        {profiles?.length === 0 && canCreate ? (
          <Button variant={open ? 'text' : 'contained'} onClick={() => setCreating(true)}>
            {t`New profile`}
          </Button>
        ) : null}
      </DialogActions>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={() => setAnchor(null)}>
        {others.map((p) => (
          <MenuItem
            key={p.id}
            onClick={() => {
              setAnchor(null)
              pick(p.id)
            }}
          >
            <ListItemIcon sx={{ color: 'inherit' }}>
              <UserRound size={16} aria-hidden={true} />
            </ListItemIcon>
            {p.name}
          </MenuItem>
        ))}
      </Menu>
      <NewProfileDialog
        open={creating}
        onClose={() => {
          setCreating(false)
          load()
        }}
      />
    </Dialog>
  )
}

// One prompt at a time; the next link's prompt follows once this one is answered.
export function ArrivalDialog() {
  const arrival = useNxm((s) => s.arrivals[0])
  return arrival ? <ArrivalPrompt key={arrival.id} arrival={arrival} /> : null
}
