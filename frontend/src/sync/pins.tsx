import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { Laptop } from 'lucide-react'
import { useEffect } from 'react'
import { create } from 'zustand'
import {
  Pins,
  SetPin,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/service.ts'
import { ICON_SIZE } from '../mods/menu.ts'
import { useSettings } from '../settings/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'

type Kind = 'mod' | 'file'

// What each synced profile keeps its own way on this computer, by "game/profile".
const usePins = create<{ byProfile: Record<string, string[] | undefined> }>(() => ({
  byProfile: {},
}))

const slot = (game: string, profile: string) => `${game}/${profile}`
const pinKey = (kind: Kind, id: string) => `${kind}:${id}`

async function loadPins(game: string, profile: string) {
  const p = await Pins(game, profile)
  const keys = [
    ...(p.mods ?? []).map((id) => pinKey('mod', id)),
    ...(p.files ?? []).map((id) => pinKey('file', id)),
  ]
  usePins.setState((s) => ({ byProfile: { ...s.byProfile, [slot(game, profile)]: keys } }))
}

function useOnlyHere(game: string, profile: string, kind: Kind, id: string) {
  const syncing = useSettings((s) => (s.syncFolder ?? '') !== '')
  const keys = usePins((s) => s.byProfile[slot(game, profile)])
  useEffect(() => {
    if (syncing && game && profile) {
      loadPins(game, profile).catch(reportUnexpected)
    }
  }, [syncing, game, profile])
  const on = keys?.includes(pinKey(kind, id)) ?? false
  const toggle = () =>
    SetPin(game, profile, kind, id, !on)
      .then(() => loadPins(game, profile))
      .catch(reportUnexpected)
  return { syncing, on, toggle }
}

/** "Only on this computer" for a mod's on/off state; absent while no sync folder is set. */
export function OnlyHereMenuItem({
  game,
  profile,
  modId,
  close,
}: {
  game: string
  profile: string
  modId: string
  close: () => void
}) {
  const { t } = useLingui()
  const { syncing, on, toggle } = useOnlyHere(game, profile, 'mod', modId)
  if (!syncing) {
    return null
  }
  return (
    <MenuAction
      icon={<Laptop size={ICON_SIZE} />}
      label={on ? t`Follow the sync again` : t`Enable or disable on this computer only`}
      onClick={() => {
        close()
        toggle().catch(reportUnexpected)
      }}
    />
  )
}

/** "Only on this computer" for one config file of a synced profile. */
export function OnlyHereFileButton({
  game,
  profile,
  file,
}: {
  game: string
  profile: string
  file: string
}) {
  const { t } = useLingui()
  const { syncing, on, toggle } = useOnlyHere(game, profile, 'file', file)
  if (!syncing) {
    return null
  }
  return (
    <Button variant={on ? 'contained' : 'outlined'} size="small" onClick={() => toggle()}>
      {t`Only on this computer`}
    </Button>
  )
}
