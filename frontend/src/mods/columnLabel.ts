import { i18n, type MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { ListColumnId } from './listColumns.ts'

const COLUMN_LABELS: Record<ListColumnId, MessageDescriptor> = {
  on: msg`On`,
  name: msg`Name`,
  version: msg`Version`,
  latest: msg`Latest`,
  released: msg`Released`,
  pluginVersion: msg`Plugin version`,
  kind: msg`Kind`,
  // Named by the game's loader where it has a term for its mods' ids (columnLabel).
  id: msg`Mod ID`,
  author: msg`Author`,
  source: msg`Source`,
  category: msg`Category`,
  endorsements: msg`Rating`,
  downloads: msg`Downloads`,
  updated: msg`Updated`,
  installed: msg`Installed`,
  switchedOff: msg`Disabled`,
  reason: msg`Install reason`,
  needs: msg`Needs`,
  status: msg`Status`,
  notes: msg`Notes and tags`,
  lastRun: msg`Last run`,
  size: msg`Size`,
  startup: msg`Startup`,
  order: msg`Order`,
}

/** idName is the loader's own term for a mod's id (UniqueID, GUID), which names that column where there is one. */
function columnLabel(id: ListColumnId, idName = ''): string {
  return id === 'id' && idName !== '' ? idName : i18n._(COLUMN_LABELS[id])
}

export { columnLabel }
