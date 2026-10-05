import { msg } from '@lingui/core/macro'
import type { State } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { savedAt, unreachable, useOffline } from './offline.ts'

// The sentence for the banner and for the tooltip of each action it disables.
function offlineMessage(down: State[], locale: string): string {
  const names = listNames(down.map((s) => sourceLabel(s.id)))
  const time = down.length === 1 ? savedAt(down[0] as State, locale) : ''
  return time
    ? i18n._(msg`${names} can't be reached; showing what Mortar saved at ${time}`)
    : i18n._(msg`${names} can't be reached; showing what Mortar saved`)
}

// The reason a network action on these sources is off, or '' when it can go ahead. With every, it is off only when
// all the listed sources that have answered lately are down.
function offlineReasonOf(states: State[], ids: string[], every = false): string {
  const known = states.filter((s) => ids.includes(s.id))
  const down = unreachable(known)
  if (down.length === 0 || (every && down.length < known.length)) {
    return ''
  }
  return offlineMessage(down, i18n.locale)
}

function useOfflineReason(ids: string[]): string {
  return offlineReasonOf(
    useOffline((s) => s.states),
    ids,
  )
}

// The sources an update check asks.
const UPDATE_SOURCES = ['nexus', 'github']

// Mod update checks go to Nexus and GitHub; they are off only when every one of them that has answered lately is down.
function updatesOfflineReason(): string {
  return offlineReasonOf(useOffline.getState().states, UPDATE_SOURCES, true)
}

function useUpdatesOfflineReason(): string {
  return offlineReasonOf(
    useOffline((s) => s.states),
    UPDATE_SOURCES,
    true,
  )
}

// The sources an update downloads from: its own, or Nexus when it names only a Nexus mod.
function updateSources(update: { source: string; nexusId: number }): string[] {
  if (update.source !== '') {
    return [update.source]
  }
  return update.nexusId > 0 ? ['nexus'] : []
}

export {
  offlineMessage,
  updateSources,
  updatesOfflineReason,
  useOfflineReason,
  useUpdatesOfflineReason,
}
