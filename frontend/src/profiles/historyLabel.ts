import { msg, plural } from '@lingui/core/macro'
import {
  HistoryChange,
  type HistoryEvent,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { i18n } from '../i18n/index.ts'

type Worded = Pick<HistoryEvent, 'change' | 'name' | 'detail' | 'count' | 'from' | 'to' | 'target'>

// historyLabel words one history event in the user's language from what the backend recorded about it.
export function historyLabel(ev: Worded): string {
  const name = ev.name ?? ''
  const detail = ev.detail ?? ''
  const count = ev.count ?? 0
  switch (ev.change) {
    case HistoryChange.ChangeAdded:
      return i18n._(msg`Added ${name}`)
    case HistoryChange.ChangeRemoved:
      return i18n._(msg`Removed ${ev.name ?? ''}`)
    case HistoryChange.ChangeUpdated: {
      const from = ev.from ?? ''
      const to = ev.to ?? ''
      return i18n._(msg`Updated ${name} from ${from} to ${to}`)
    }
    case HistoryChange.ChangeEnabled:
      return i18n._(msg`Enabled ${ev.name ?? ''}`)
    case HistoryChange.ChangeDisabled:
      return i18n._(msg`Disabled ${{ names: name }}`)
    case HistoryChange.ChangePinned:
      return i18n._(msg`Pinned ${name} at its version`)
    case HistoryChange.ChangeUnpinned:
      return i18n._(msg`Unpinned ${name}`)
    case HistoryChange.ChangeTagged:
      return i18n._(msg`Tagged ${name} with ${detail}`)
    case HistoryChange.ChangeUntagged:
      return i18n._(msg`Removed the tag ${detail} from ${name}`)
    case HistoryChange.ChangeTags:
      return i18n._(msg`Changed the tags of ${name}`)
    case HistoryChange.ChangeNote:
      return i18n._(msg`Edited the note of ${name}`)
    case HistoryChange.ChangeMods:
      return count > 1
        ? plural(count, { one: 'Changed # mod', other: 'Changed # mods' })
        : i18n._(msg`Changed mods`)
    case HistoryChange.ChangeImported:
      return plural(count, { one: 'Imported # mod', other: 'Imported # mods' })
    case HistoryChange.ChangeMoved:
      return plural(count, {
        one: "Moved # mod from the game's Mods folder",
        other: "Moved # mods from the game's Mods folder",
      })
    case HistoryChange.ChangeRestored:
      return plural(count, { one: 'Restored # mod', other: 'Restored # mods' })
    case HistoryChange.ChangeRestoredFromStore:
      return name === ''
        ? plural(count, {
            one: 'Restored # mod from the store',
            other: 'Restored # mods from the store',
          })
        : i18n._(msg`Restored ${name} from the store`)
    case HistoryChange.ChangeBeforeEdit:
      return i18n._(msg`Before this change`)
    case HistoryChange.ChangeReverted: {
      // A relative time would read "Went back to 2 minutes ago"; the point gone back to is a date.
      const when = formatWhen(ev.target ?? '', { withTime: true, absolute: true })
      return i18n._(msg`Went back to ${when}`)
    }
    case HistoryChange.ChangeRestoredFile:
      return i18n._(msg`Restored ${detail} of ${name}`)
    case HistoryChange.ChangeConfigEdited:
      return i18n._(msg`Edited the settings of ${name}`)
    case HistoryChange.ChangeConfigReset:
      return i18n._(msg`Reset the settings of ${name}`)
    case HistoryChange.ChangePresetApplied:
      return i18n._(msg`Applied the preset ${detail} to ${name}`)
    case HistoryChange.ChangeOptionSet:
      return i18n._(msg`Set ${detail} of ${name} for the next start`)
    case HistoryChange.ChangeCategoryRemoved:
      return i18n._(msg`Deleted a custom category`)
    case HistoryChange.ChangeChannel:
      return i18n._(msg`Switched ${name} to the ${detail} update channel`)
    case HistoryChange.ChangeCollectionUnlinked:
      return name === ''
        ? i18n._(msg`Unlinked the collection`)
        : i18n._(msg`Unlinked the collection ${name}`)
    case HistoryChange.ChangeTrimmed:
      return plural(count, {
        one: 'Trimmed history, dropped # older change',
        other: 'Trimmed history, dropped # older changes',
      })
    case HistoryChange.ChangeUnscanned:
      return i18n._(msg`Installed ${name} although the antivirus flagged it: ${detail}`)
    case HistoryChange.ChangeKnownGood:
      return i18n._(msg`Known good`)
    case HistoryChange.ChangeGroups:
      return i18n._(msg`Changed groups`)
    case HistoryChange.ChangeLoader:
      return i18n._(msg`Changed loader`)
    case HistoryChange.ChangeInstall:
      return i18n._(msg`Changed game install`)
    case HistoryChange.ChangeSaves:
      return i18n._(msg`Changed separate saves`)
    case HistoryChange.ChangeLaunch:
      return i18n._(msg`Changed launch settings`)
    case HistoryChange.ChangeSettings:
      return i18n._(msg`Changed profile settings`)
    default:
      return i18n._(msg`Changed the profile`)
  }
}
