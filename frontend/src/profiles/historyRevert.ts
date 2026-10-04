import { msg } from '@lingui/core/macro'
import type {
  HistoryEvent,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  History,
  Revert,
  Snapshot,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
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
} from '../toasts/undo.ts'
import { useProfiles } from './store.ts'

interface RevertOutcome {
  events: HistoryEvent[]
  error: string
  missingEvent: string
  missingNames: string[]
  missingWants: RevertWant[]
}

const emptyMissing = {
  missingEvent: '',
  missingNames: [] as string[],
  missingWants: [] as RevertWant[],
}

async function missingFromError(
  game: string,
  profileId: string,
  eventId: string,
  message: string,
): Promise<Pick<RevertOutcome, 'error' | 'missingEvent' | 'missingNames' | 'missingWants'>> {
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
    }
  } catch {
    return { error: message, missingEvent: eventId, missingNames: listed, missingWants: [] }
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
