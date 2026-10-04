import { useLingui } from '@lingui/react/macro'
import { useEffect, useRef, useState } from 'react'
import { Share } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { toastError } from '../toasts/report.ts'
import { type ShownInfo, shownInfo, suggestFile } from './logic.ts'
import { shareIncludeDefaults, toShareInclude } from './shareDefaults.ts'
import { useShareDialog } from './store.ts'

type Tab = 'link' | 'file'

export function useShareBuild() {
  const { t } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const keys = useShareDialog((s) => s.keys)
  const close = useShareDialog((s) => s.close)
  const game = useProfiles((s) => s.game?.id ?? 'stardew')
  const [tab, setTab] = useState<Tab>('link')
  const [info, setInfo] = useState<ShownInfo | null>(null)
  const [include, setInclude] = useState(() => shareIncludeDefaults(useSettings.getState()))
  const pickTab = useRef(true)
  useEffect(() => {
    if (!profileId) {
      return
    }
    setInclude(shareIncludeDefaults(useSettings.getState()))
    setTab('link')
    setInfo(null)
    pickTab.current = true
  }, [profileId])
  useEffect(() => {
    if (!profileId) {
      return
    }
    let stale = false
    Share(game, profileId, keys, toShareInclude(include)).then(
      (next) => {
        if (!stale) {
          setInfo(shownInfo(next))
          if (pickTab.current) {
            pickTab.current = false
            setTab(suggestFile(next.count, next.tooLarge) ? 'file' : 'link')
          }
        }
      },
      (e: unknown) => {
        if (stale) {
          return
        }
        toastError(t`Could not build the link`, e)
        close()
      },
    )
    return () => {
      stale = true
    }
  }, [profileId, keys, game, close, include, t])
  return { profileId, keys, close, game, tab, setTab, info, include, setInclude }
}
