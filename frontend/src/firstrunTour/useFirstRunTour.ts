import { useLingui } from '@lingui/react/macro'
import { useCallback, useEffect, useLayoutEffect, useState } from 'react'
import { SetTipsSeen } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportError } from '../toasts/report.ts'
import { anyModalOpen, resolveTourAnchor } from './anchors.ts'
import { sameRectOr, type TourRect, tourEligible } from './logic.ts'
import { useTourReplay } from './replay.ts'
import { tourMarkSeen, tourShouldRun } from './seen.ts'

const ANCHOR_POLL_MS = 250

function useFirstRunTour() {
  const { t } = useLingui()
  const seen = useSettings((s) => s.tipsSeen)
  const route = useNav((s) => s.route)
  const loaded = useProfiles((s) => s.loaded)
  const openId = useProfiles((s) => s.openId)
  const hasProfile = useProfiles((s) => s.profiles.some((p) => p.id === openId))
  const replay = useTourReplay((s) => s.pending)
  const clearReplay = useTourReplay((s) => s.clear)
  const [open, setOpen] = useState(false)
  const [step, setStep] = useState(0)
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const [anchorRect, setAnchorRect] = useState<TourRect | null>(null)
  // Closing must not wait for the saved setting to round-trip, or a failed save reopens the tour.
  const [dismissed, setDismissed] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)

  // MUI keeps no open-dialog store, so watch the DOM; the tour waits while one is open.
  useEffect(() => {
    const sync = () => setModalOpen(anyModalOpen())
    sync()
    const id = globalThis.setInterval(sync, ANCHOR_POLL_MS)
    return () => globalThis.clearInterval(id)
  }, [])

  const onGameWithProfile = route.name === 'game' && loaded && openId !== '' && hasProfile
  const eligible = tourEligible({
    onGameWithProfile,
    unseen: tourShouldRun(seen),
    dismissed,
    replay,
    modalOpen,
  })

  const finish = useCallback(() => {
    setDismissed(true)
    clearReplay()
    setOpen(false)
    SetTipsSeen(tourMarkSeen(seen)).catch(reportError(t`Could not save that setting`))
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
      setAnchorRect(null)
      return
    }
    const sync = () => {
      const el = resolveTourAnchor(step)
      setAnchorEl(el)
      setAnchorRect((prev) => sameRectOr(prev, el?.getBoundingClientRect() ?? null))
    }
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

  return { active, anchorEl, anchorRect, step, setStep, finish }
}

export { useFirstRunTour }
