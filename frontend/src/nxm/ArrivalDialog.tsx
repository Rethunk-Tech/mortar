import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Menu,
  MenuItem,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type { Arrival } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { List } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { paper } from '../mods/paper.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
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
  const dismiss = useNxm((s) => s.dismiss)
  const lastId = useSettings((s) => s.lastProfile?.[NXM_GAME])
  const name = useModName(arrival.link.modId)
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  useEffect(() => {
    List(NXM_GAME)
      .then((list) => setProfiles((list ?? []).filter((p) => !p.hidden)))
      .catch(reportUnexpected)
  }, [])
  const open = profiles.find((p) => p.id === lastId) ?? profiles[0]
  const others = profiles.filter((p) => p.id !== open?.id)
  return (
    <Dialog
      open={true}
      onClose={(_, reason) => {
        if (reason === 'escapeKeyDown') {
          dismiss(arrival.id)
        }
      }}
      slotProps={{ paper: { sx: { ...paper.sx, width: 460, maxWidth: 'calc(100% - 32px)' } } }}
    >
      <DialogTitle>{t`Install ${name}?`}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {t`You started this download on Nexus. Choose the profile it goes into.`}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => dismiss(arrival.id)}>{t`Ignore`}</Button>
        {others.length > 0 ? (
          <Button variant="outlined" onClick={(e) => setAnchor(e.currentTarget)}>
            {t`Other profile…`}
          </Button>
        ) : null}
        {open ? (
          <Button variant="contained" onClick={() => choose(arrival.id, open.id)}>
            {open.name}
          </Button>
        ) : null}
      </DialogActions>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={() => setAnchor(null)}>
        {others.map((p) => (
          <MenuItem
            key={p.id}
            onClick={() => {
              setAnchor(null)
              choose(arrival.id, p.id)
            }}
          >
            {p.name}
          </MenuItem>
        ))}
      </Menu>
    </Dialog>
  )
}

// One prompt at a time; the next link's prompt follows once this one is answered.
export function ArrivalDialog() {
  const arrival = useNxm((s) => s.arrivals[0])
  return arrival ? <ArrivalPrompt key={arrival.id} arrival={arrival} /> : null
}
