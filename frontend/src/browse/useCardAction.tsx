import { useLingui } from '@lingui/react/macro'
import { type ReactNode, useEffect, useRef, useState } from 'react'
import { listedTitle } from '../mods/brokenKind.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { usePending } from '../toasts/usePending.ts'
import { GITHUB, NEXUS, THUNDERSTORE } from './browseConstants.ts'
import type { BrowseItem, ResultCardProps } from './browseTypes.ts'
import { CardAction } from './CardAction.tsx'
import { cardState, inProfile, isActive, isInstalled, shownState } from './cardState.ts'
import { pickedItem } from './pickedItem.ts'

type ActionProps = Pick<
  ResultCardProps,
  'premium' | 'openUrl' | 'downloadNexus' | 'addGitHub' | 'addPackage' | 'addDirect' | 'profileID'
>

interface CardActionState {
  // The hit as the picked source sees it.
  shownItem: BrowseItem
  url: string
  installed: boolean
  // Add (or download, or open the files page) from the picked source, as the action button does.
  // then runs once the install starts: right away, or after the player accepts a known-broken warning.
  add: (then?: () => void) => void
  action: ReactNode
  // The known-broken warning, rendered outside the card so its events do not reach the card's click.
  dialog: ReactNode
}

/** The Add, Download or Open files page action of a hit from source, shared by its card and its details panel. */
function useCardAction(props: ActionProps, item: BrowseItem, source: string): CardActionState {
  const { t } = useLingui()
  const { premium, openUrl, profileID } = props
  const [pending, run] = usePending()
  // The install a known-broken hit's Add holds until the player answers.
  const [asking, setAsking] = useState<(() => void) | null>(null)
  const [openedFiles, setOpenedFiles] = useState(false)
  const items = useQueue((s) => s.state.items)
  const primary = { source: item.source, id: item.id, url: item.url, installed: item.installed }
  const choices = [primary, ...(item.alts ?? [])]
  const { id, url } = choices.find((c) => c.source === source) ?? primary
  const shownItem = pickedItem(item, source)
  // The profile's own entries are the live word: the search result's flag is only as new as the search, so it counts
  // only until the profile changes.
  const profile = useProfiles(openProfileOf)
  const searchedUpdated = useRef(profile?.updated)
  const held = inProfile(profile, source, id)
  const installed = isInstalled(
    held,
    choices.some((c) => c.installed),
    profile?.updated === searchedUpdated.current,
  )
  const live = cardState(items, source, id, profileID)
  // A free account's click on Mod Manager Download is not ours to see; the card waits for its nxm item.
  const queueState =
    live.kind === 'idle' && openedFiles ? ({ kind: 'waiting-nexus' } as const) : live
  const [watched, setWatched] = useState(false)
  useEffect(() => {
    if (isActive(queueState)) {
      setWatched(true)
    }
  }, [queueState])
  // A mod removed again (Undo from the bell) is no longer one this card watched arrive.
  const wasHeld = useRef<boolean>(false)
  useEffect(() => {
    if (held) {
      wasHeld.current = true
    } else if (wasHeld.current) {
      wasHeld.current = false
      setWatched(false)
    }
  }, [held])
  const shown = shownState(queueState, installed, watched)
  const addFromSource = () => {
    if (source === GITHUB) {
      return props.addGitHub(id)
    }
    return source === THUNDERSTORE
      ? props.addPackage(id)
      : props.addDirect(source, id, shownItem.picture)
  }
  // A hit on the game's known-broken list asks before it installs.
  const askFirst = (install: () => void) => (then?: () => void) => {
    const go = () => {
      install()
      then?.()
    }
    return shownItem.knownBroken ? setAsking(() => go) : go()
  }
  const onAdd = askFirst(() => run(() => Promise.resolve(addFromSource())))
  const onDownload = askFirst(() => run(() => Promise.resolve(props.downloadNexus(id))))
  const onOpenFiles = askFirst(() => {
    setOpenedFiles(true)
    openUrl(`${url}?tab=files`)
  })
  const listed = shownItem.knownBroken ?? undefined
  const add = (then?: () => void) => {
    if (source !== NEXUS) {
      onAdd(then)
    } else if (premium) {
      onDownload(then)
    } else {
      onOpenFiles(then)
    }
  }
  return {
    shownItem,
    url,
    installed,
    add,
    action: (
      <CardAction
        item={shownItem}
        source={source}
        state={shown}
        installed={installed}
        premium={premium}
        pending={pending}
        onAdd={() => onAdd()}
        onDownload={() => onDownload()}
        onOpenFiles={() => onOpenFiles()}
      />
    ),
    dialog: (
      <ConfirmationDialog
        open={asking !== null && listed !== undefined}
        title={listedTitle(listed?.kind, shownItem.name)}
        body={
          listed?.replacement
            ? t`${{ reason: listed.reason }} Use ${{ replacement: listed.replacement }} instead.`
            : listed?.reason
        }
        confirmLabel={t`Install anyway`}
        tone="warning"
        onCancel={() => setAsking(null)}
        onConfirm={() => {
          asking?.()
          setAsking(null)
        }}
      />
    ),
  }
}

export { type ActionProps, useCardAction }
