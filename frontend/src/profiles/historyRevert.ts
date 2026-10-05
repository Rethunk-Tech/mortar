import { msg } from '@lingui/core/macro'
import type {
  HistoryEvent,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  History,
  Revert,
  Snapshot,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { download } from '../queue/actions.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  downloadWantsForEntries,
  entriesMatchingMissing,
  missingModNames,
  parseMissingStoreList,
  type RevertWant,
  type UndoEntry,
  undoRevertTarget,
  unfetchableNames,
} from '../toasts/undo.ts'
import { useProfiles } from './store.ts'

interface RevertOutcome {
  events: HistoryEvent[]
  error: string
  missingEvent: string
  missingNames: string[]
  missingWants: RevertWant[]
  // Missing mods with no source to download them from again.
  missingUnfetchable: string[]
  // The revert failed on missing mods, but the change's record could not be read to name them.
  missingUnread: boolean
}

const emptyMissing = {
  missingEvent: '',
  missingNames: [] as string[],
  missingWants: [] as RevertWant[],
  missingUnfetchable: [] as string[],
  missingUnread: false,
}

/** Says in a toast which mods a failed revert lacks, for a caller with no missing-mods UI of its own. */
function pushMissingToast(outcome: RevertOutcome) {
  if (outcome.missingUnread) {
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(
        msg`Could not undo: some mods are missing, and the change's record could not be read.`,
      ),
    })
    return
  }
  if (outcome.missingNames.length === 0) {
    return
  }
  const list = outcome.missingNames.join(', ')
  const wants = outcome.missingWants
  useToasts.getState().push({
    kind: 'error',
    title: i18n._(msg`Could not undo. Missing from the store: ${list}`),
    ...(outcome.missingUnfetchable.length > 0
      ? {
          body: i18n._(msg`Mortar cannot download ${outcome.missingUnfetchable.join(', ')} again.`),
        }
      : {}),
    ...(wants.length > 0
      ? { action: { label: i18n._(msg`Download missing`), run: () => download(wants) } }
      : {}),
  })
}

async function missingFromError(
  game: string,
  profileId: string,
  eventId: string,
  message: string,
): Promise<Omit<RevertOutcome, 'events'>> {
  const listed = parseMissingStoreList(message)
  if (listed.length === 0) {
    return { error: message, ...emptyMissing }
  }
  try {
    const snap = ((await Snapshot(game, profileId, eventId)) ?? []) as UndoEntry[]
    const hit = entriesMatchingMissing(snap, listed)
    const names = missingModNames(hit)
    return {
      error: message,
      missingEvent: eventId,
      missingNames: names.length > 0 ? names : listed,
      missingWants: downloadWantsForEntries(hit),
      missingUnfetchable: unfetchableNames(hit),
      missingUnread: false,
    }
  } catch {
    return { error: message, ...emptyMissing, missingEvent: eventId, missingUnread: true }
  }
}

function pushRevertUndo(game: string, profileId: string, beforeId: string, next: Profile) {
  useToasts.getState().push({
    kind: 'success',
    title: i18n._(msg`Restored ${next.name} to before that change`),
    action: {
      label: i18n._(msg`Undo`),
      profileId,
      run: async () => {
        useProfiles.getState().replace(await Revert(game, profileId, beforeId))
      },
      live: () => {
        const current = useProfiles.getState().profiles.find((p) => p.id === profileId)
        if (!current) {
          return { disabled: true, reason: i18n._(msg`That profile is gone.`) }
        }
        const keys = new Set((next.entries ?? []).map((entry) => entry.key))
        const now = new Set((current.entries ?? []).map((entry) => entry.key))
        if (keys.size !== now.size || [...keys].some((key) => !now.has(key))) {
          return { disabled: true, reason: i18n._(msg`This is no longer the latest change.`) }
        }
        return { disabled: false }
      },
    },
  })
}

export async function revertHistoryEvent(
  game: string,
  profileId: string,
  eventId: string,
  events: HistoryEvent[],
): Promise<RevertOutcome> {
  const beforeId = undoRevertTarget(events)
  try {
    const next = await Revert(game, profileId, eventId)
    useProfiles.getState().replace(next)
    if (beforeId !== '') {
      pushRevertUndo(game, profileId, beforeId, next)
    }
    return { events: (await History(game, profileId)) ?? [], error: '', ...emptyMissing }
  } catch (e) {
    return {
      events,
      ...(await missingFromError(game, profileId, eventId, errorMessage(e))),
    }
  }
}

export { pushMissingToast }
