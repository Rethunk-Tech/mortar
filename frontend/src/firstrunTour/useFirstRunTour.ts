import { useLingui } from '@lingui/react/macro'
import { useCallback, useEffect, useLayoutEffect, useState } from 'react'
import { SetTipsSeen } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { errorText } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { resolveTourAnchor } from './anchors.ts'
import { useTourReplay } from './replay.ts'
import { tourMarkSeen, tourShouldRun } from './seen.ts'

const ANCHOR_POLL_MS = 250

function useFirstRunTour() {
  const { t } = useLingui()
  const seen = useSettings((s) => s.tipsSeen)
  const route = useNav((s) => s.route)
  const loaded = useProfiles((s) => s.loaded)
  const openId = useProfiles((s) => s.openId)
  const hasProfile = useProfiles((s) => s.profiles.some((p) => p.id === openId && !p.hidden))
  const replay = useTourReplay((s) => s.pending)
  const clearReplay = useTourReplay((s) => s.clear)
  const [open, setOpen] = useState(false)
  const [step, setStep] = useState(0)
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)

  const onGameWithProfile = route.name === 'game' && loaded && openId !== '' && hasProfile
  const eligible = onGameWithProfile && (tourShouldRun(seen) || replay)

  const finish = useCallback(() => {
    clearReplay()
    setOpen(false)
    SetTipsSeen(tourMarkSeen(seen)).catch((err: unknown) => {
      const body = errorText(err)
      useToasts.getState().push({
        kind: 'error',
        title: t`Couldn't save that setting`,
        ...(body ? { body } : {}),
      })
    })
  }, [clearReplay, seen, t])

  useEffect(() => {
    if (!eligible || open) {
      return
    }
    setStep(0)
    setOpen(true)
  }, [eligible, open])

  useLayoutEffect(() => {
    if (!open) {
      setAnchorEl(null)
      return
    }
    const sync = () => setAnchorEl(resolveTourAnchor(step))
    sync()
    const id = globalThis.setInterval(sync, ANCHOR_POLL_MS)
    globalThis.addEventListener('resize', sync)
    globalThis.addEventListener('scroll', sync, true)
    return () => {
      globalThis.clearInterval(id)
      globalThis.removeEventListener('resize', sync)
      globalThis.removeEventListener('scroll', sync, true)
    }
  }, [open, step])

  useEffect(() => {
    if (!open) {
      return
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        finish()
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [open, finish])

  const active = open && eligible && anchorEl !== null

  return { active, anchorEl, step, setStep, finish }
}

export { useFirstRunTour }
