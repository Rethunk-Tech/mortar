import { i18n, type MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { ListColumnId } from './listColumns.ts'

const COLUMN_LABELS: Record<ListColumnId, MessageDescriptor> = {
  on: msg`On`,
  name: msg`Name`,
  version: msg`Version`,
  latest: msg`Latest on Nexus`,
  id: msg`UniqueID`,
  author: msg`Author`,
  source: msg`Source`,
  category: msg`Category`,
  endorsements: msg`Endorsements`,
  downloads: msg`Downloads`,
  updated: msg`Updated on Nexus`,
  installed: msg`Installed`,
  needs: msg`Needs`,
  status: msg`Status`,
  notes: msg`Notes and tags`,
  lastRun: msg`Last run`,
  size: msg`Size`,
  startup: msg`Startup`,
  order: msg`Order`,
}

function columnLabel(id: ListColumnId): string {
  return i18n._(COLUMN_LABELS[id])
}

export { columnLabel }
