import { useLingui } from '@lingui/react/macro'
import { useEffect, useRef, useState } from 'react'
import { Share } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { hasThunderstore } from '../profiles/packImport.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { toastError } from '../toasts/report.ts'
import { type ShownInfo, shownInfo, suggestFile } from './logic.ts'
import {
  isShareMethod,
  methodStorageKey,
  pickMethod,
  type ShareMethod,
  shareMethods,
} from './methods.ts'
import { shareIncludeDefaults, toShareInclude } from './shareDefaults.ts'
import { useShareDialog } from './store.ts'

export function useShareBuild() {
  const { t } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const keys = useShareDialog((s) => s.keys)
  const close = useShareDialog((s) => s.close)
  const game = useCurrentGame()
  const thunderstore = useProfiles((s) => hasThunderstore(s.game))
  const [method, setMethodState] = useState<ShareMethod>('link')
  const setMethod = (next: ShareMethod) => {
    setMethodState(next)
    writeStored(methodStorageKey(game), next)
  }
  const [info, setInfo] = useState<ShownInfo | null>(null)
  const [include, setInclude] = useState(() => shareIncludeDefaults(useSettings.getState()))
  const pickTab = useRef(true)
  useEffect(() => {
    if (!profileId) {
      return
    }
    setInclude(shareIncludeDefaults(useSettings.getState()))
    setMethodState('link')
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
            const remembered = readStored<ShareMethod | null>(
              methodStorageKey(game),
              null,
              (v): v is ShareMethod => isShareMethod(v),
            )
            setMethodState(
              pickMethod(
                remembered,
                suggestFile(next.count, next.tooLarge) ? 'file' : 'link',
                shareMethods({ thunderstore, count: next.count }),
              ),
            )
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
  }, [profileId, keys, game, close, include, t, thunderstore])
  return {
    profileId,
    keys,
    close,
    game,
    method,
    setMethod,
    thunderstore,
    info,
    include,
    setInclude,
  }
}
