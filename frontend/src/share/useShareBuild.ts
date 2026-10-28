import { useLingui } from '@lingui/react/macro'
import { useEffect, useRef, useState } from 'react'
import type { Saved } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import {
  SaveFile,
  Share,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { sameValue } from '../shell/sameValue.ts'
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { type ShownInfo, shownInfo, suggestFile } from './logic.ts'
import {
  type Destination,
  destinationStorageKey,
  isDestination,
  lastUsedDestination,
  type MortarFormat,
  shareDestinations,
} from './methods.ts'
import { shareIncludeDefaults, toShareInclude } from './shareDefaults.ts'
import { useShareDialog } from './store.ts'

export function useShareBuild() {
  const { t } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const keys = useShareDialog((s) => s.keys)
  const close = useShareDialog((s) => s.close)
  const game = useCurrentGame()
  const thunderstore = useProfiles((s) => s.game?.sources?.includes('thunderstore') === true)
  // null is the destination grid.
  const [destination, setDestination] = useState<Destination | null>(null)
  const [format, setFormat] = useState<MortarFormat>('link')
  const [lastUsed, setLastUsed] = useState<Destination | null>(null)
  const choose = (next: Destination) => {
    setDestination(next)
    writeStored(destinationStorageKey(game), next)
  }
  const [info, setInfo] = useState<ShownInfo | null>(null)
  const [include, setInclude] = useState(() => shareIncludeDefaults(useSettings.getState()))
  const pickTab = useRef<boolean>(true)
  useEffect(() => {
    if (!profileId) {
      return
    }
    // An equal set keeps its object, so opening the dialog builds the share once, not again for the reset.
    const defaults = shareIncludeDefaults(useSettings.getState())
    setInclude((cur) => (sameValue(cur, defaults) ? cur : defaults))
    setDestination(null)
    setFormat('link')
    setLastUsed(null)
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
            const remembered = readStored<Destination | null>(
              destinationStorageKey(game),
              null,
              (v): v is Destination => isDestination(v),
            )
            setLastUsed(
              lastUsedDestination(
                remembered,
                shareDestinations({
                  thunderstore,
                  count: next.count,
                  leftOut: next.leftOut?.length ?? 0,
                }),
              ),
            )
            setFormat(suggestFile(next.count, next.tooLarge) ? 'file' : 'link')
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
  // The .mortar file save: the dialog's footer starts it and the File view shows where it went.
  const [saved, setSaved] = useState<Saved | null>(null)
  const [saving, runSave] = usePending()
  const saveFile = () =>
    runSave(
      async () => {
        const result = await SaveFile(game, profileId, keys, toShareInclude(include))
        if (result.path) {
          setSaved(result)
          useToasts.getState().push({ kind: 'success', title: t`File saved` })
        }
      },
      { errorTitle: t`Could not save the file` },
    )
  return {
    saved,
    saving,
    saveFile,
    profileId,
    keys,
    close,
    game,
    destination,
    setDestination,
    choose,
    format,
    setFormat,
    lastUsed,
    thunderstore,
    info,
    include,
    setInclude,
  }
}
