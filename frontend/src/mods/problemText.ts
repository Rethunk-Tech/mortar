import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { i18n } from '../i18n/index.ts'
import { useDescribe, useDescribeDrift } from './describe.ts'
import type { DismissedRow, ProblemSectionId, Row } from './problemGroups.ts'

// What is broken, as against advice such as harmless overlaps or cleanup.
const ERROR_SECTIONS = new Set<string>([
  'missing',
  'broken',
  'damaged',
  'runErrors',
  'loadFailures',
  'pluginClashes',
])

const isDismissedRow = (row: Row | DismissedRow): row is DismissedRow => 'row' in row

function useSectionTitle() {
  const { t } = useLingui()
  return (id: ProblemSectionId) => {
    switch (id) {
      case 'missing':
        return t`Missing requirements`
      case 'conflicts':
        return t`Conflicts`
      case 'broken':
        return t`Broken or outdated mods`
      case 'damaged':
        return t`Damaged`
      case 'runErrors':
        return t`Last run`
      case 'loadFailures':
        return t`Did not load`
      case 'pluginClashes':
        return t`Duplicate plugins`
      case 'drift':
        return t`Changed`
      case 'duplicates':
        return t`Duplicates`
      case 'deprecated':
        return t`Deprecated`
      case 'settings':
        return t`Settings`
      case 'cosmetic':
        return t`Cosmetic`
      case 'dismissed':
        return t`Dismissed`
      default:
        return ''
    }
  }
}

// useRowText is a row's sentence and the author's note shown under it, shared by the list and Copy all.
function useRowText() {
  const describe = useDescribe()
  const describeDrift = useDescribeDrift()
  return (row: Row) => {
    let note = ''
    if (row.kind === 'missing' && row.missing.listed) {
      note = row.missing.note.trim()
    } else if (row.kind === 'setting') {
      note = row.setting.description.trim()
    } else if (row.kind === 'loadFailure') {
      note = loadKindText(row.loadFailure.kind)
    }
    let text = row.kind === 'drift' ? describeDrift(row.drift) : describe(row)
    if (note !== '' && text.endsWith(`: ${note}`)) {
      text = text.slice(0, -(note.length + 2))
    }
    return { text, note }
  }
}

function loadKindText(kind: string): string {
  switch (kind) {
    case 'missing-dependency':
      return i18n._(msg`Missing dependency`)
    case 'incompatible-version':
      return i18n._(msg`Incompatible version`)
    case 'load-exception':
      return i18n._(msg`Error while loading`)
    case 'patch-exception':
      return i18n._(msg`Error in a patch`)
    case 'preloader-patch':
      return i18n._(msg`Error in a preloader patch`)
    case 'game-version':
      return i18n._(msg`Game version not supported`)
    case 'unity-exception':
      return i18n._(msg`Unity exception`)
    case 'dependency-not-loaded':
      return i18n._(msg`A plugin it needs failed`)
    case 'duplicate-guid':
      return i18n._(msg`Older copy skipped`)
    case 'incompatible-plugin':
      return i18n._(msg`Incompatible with another plugin`)
    case 'loader-version':
      return i18n._(msg`Needs a newer loader`)
    case 'chainloader':
      return i18n._(msg`Loader failed to start`)
    case 'game-setting':
      return i18n._(msg`Game setting off`)
    case 'run-failed':
      return i18n._(msg`The game never started`)
    case 'plugin-error':
      return i18n._(msg`Plugin reported an error`)
    default:
      return kind
  }
}

export { ERROR_SECTIONS, isDismissedRow, loadKindText, useRowText, useSectionTitle }
