import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { PrefCopy } from './prefCopyAntivirus.ts'

// Settings › Appearance › Mods view.
export function modsViewPrefs(i18n: I18n): Record<string, PrefCopy> {
  return {
    defaultModsView: {
      label: i18n._(msg`Default mods view`),
      description: i18n._(msg`Grid or list for new sessions`),
      options: [
        { value: 'grid', label: i18n._(msg`Grid`) },
        { value: 'list', label: i18n._(msg`List`) },
      ],
    },
    implicitModsView: {
      label: i18n._(msg`Implicit mods`),
      description: i18n._(msg`How mods Mortar brought in for other mods appear in the mod list`),
      options: [
        { value: 'shown', label: i18n._(msg`Always shown`) },
        { value: 'nested', label: i18n._(msg`Nested`) },
        { value: 'hidden', label: i18n._(msg`Hidden`) },
      ],
    },
    gridCardSize: {
      label: i18n._(msg`Grid card size`),
      options: [
        { value: 'small', label: i18n._(msg`Small`) },
        { value: 'medium', label: i18n._(msg`Medium`) },
        { value: 'large', label: i18n._(msg`Large`) },
      ],
    },
    showAuthorOnCards: {
      label: i18n._(msg`Author on cards`),
      description: i18n._(msg`Show the author line on grid cards`),
    },
    centerConfigControls: {
      label: i18n._(msg`Centre config controls`),
      description: i18n._(
        msg`On a wide window, each config setting's control sits mid-row, nearer its name`,
      ),
    },
    listGroupBy: {
      label: i18n._(msg`Default grouping`),
      options: [
        { value: 'none', label: i18n._(msg`None`) },
        { value: 'status', label: i18n._(msg`Status`) },
        { value: 'category', label: i18n._(msg`Category`) },
        { value: 'source', label: i18n._(msg`Source`) },
        { value: 'tag', label: i18n._(msg`Tag`) },
        { value: 'framework', label: i18n._(msg`Framework`) },
        { value: 'download', label: i18n._(msg`Download`) },
        { value: 'author', label: i18n._(msg`Author`) },
        { value: 'group', label: i18n._(msg`Group`) },
        { value: 'reason', label: i18n._(msg`Install reason`) },
      ],
    },
  }
}
