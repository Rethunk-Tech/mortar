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
import type { Arrival } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { List } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { formatKb } from '../i18n/bytes.ts'
import { useNexusDetails } from '../mods/nexusDetails.ts'
import { useProfiles } from '../profiles/store.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { NXM_GAME } from './route.ts'
import { fallbackName, modName, useNxm } from './store.ts'

function useModName(modId: number): string {
  const [name, setName] = useState(() => fallbackName(modId))
  useEffect(() => {
    let live = true
    setName(fallbackName(modId))
    modName(modId)
      .then((n) => live && setName(n))
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [modId])
  return name
}

function ArrivalPrompt({ arrival }: { arrival: Arrival }) {
  const { t } = useLingui()
  const choose = useNxm((s) => s.choose)
  const arrivals = useNxm((s) => s.arrivals)
  const dismiss = useNxm((s) => s.dismiss)
  const lastId = useSettings((s) => gamePrefs(s).nxmDefaultProfile || s.lastProfile?.[NXM_GAME])
  const name = useModName(arrival.link.modId)
  const file = useNexusDetails((s) =>
    s.byId[arrival.link.modId]?.details?.files?.find((f) => f.fileId === arrival.link.fileId),
  )
  // New profile creates in the game the profiles store has open, so it is offered only when that is this game.
  const canCreate = useProfiles((s) => s.game?.id === NXM_GAME)
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const [profiles, setProfiles] = useState<Profile[] | null>(null)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [creating, setCreating] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [applyAll, setApplyAll] = useState(false)
  const load = useCallback(() => {
    List(NXM_GAME)
      .then((list) => setProfiles((list ?? []).filter((p) => !p.hidden)))
      .catch((e: unknown) => {
        setProfiles([])
        reportUnexpected(e)
      })
  }, [])
  useEffect(load, [load])
  const open = profiles?.find((p) => p.id === lastId) ?? profiles?.[0]
  const others = profiles?.filter((p) => p.id !== open?.id) ?? []
  const pick = (profile: string) => {
    if (busy) {
      return
    }
    setBusy(true)
    setError('')
    const ids = applyAll ? arrivals.map((a) => a.id) : [arrival.id]
    Promise.all(ids.map((id) => choose(id, profile)))
      .catch((e: unknown) => setError(errorMessage(e)))
      .finally(() => setBusy(false))
  }
  let text = t`You started this download on Nexus. Choose the profile it goes into.`
  if (profiles === null) {
    text = t`Loading profiles…`
  } else if (profiles.length === 0) {
    text = canCreate
      ? t`You started this download on Nexus. There is no profile to put it in yet; create one first.`
      : t`You started this download on Nexus, but there is no ${gameName} profile to put it in yet.`
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
          <DialogContentText color="error" sx={{ mt: 1 }}>
            {error}
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
            variant="outlined"
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
          <Button variant="contained" onClick={() => setCreating(true)}>
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
