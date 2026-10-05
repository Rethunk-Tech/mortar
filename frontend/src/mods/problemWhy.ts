import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
import type { Row } from './problemGroups.ts'

const GUIDE = 'https://github.com/Rethunk-Tech/mortar/blob/main/docs/user-guide.md'

interface Why {
  anchor: string
  text: () => string
}

// One plain paragraph per kind of row, and the heading of docs/user-guide.md that expands on it. The record is
// keyed by every Row kind, so a new kind does not compile until it has both.
export const problemWhy: Record<Row['kind'], Why> = {
  missing: {
    anchor: 'missing-requirement',
    text: () =>
      i18n._(
        msg`A mod you enabled needs another mod that is not in this profile, or is disabled. The first mod may fail to load or leave out features until the other one is added.`,
      ),
  },
  broken: {
    anchor: 'broken-mod',
    text: () =>
      i18n._(
        msg`SMAPI's list marks this mod as broken, obsolete or abandoned for your game version. It may crash, do nothing, or stop working after the next game update.`,
      ),
  },
  duplicate: {
    anchor: 'duplicate-mod',
    text: () =>
      i18n._(
        msg`Two enabled copies of the same mod are in this profile. The game loads only one of them, and you do not choose which.`,
      ),
  },
  asset: {
    anchor: 'conflicting-files',
    text: () =>
      i18n._(
        msg`Two mods change the same file. Only one change wins, so the other mod may look or behave wrongly.`,
      ),
  },
  runError: {
    anchor: 'errors-in-the-last-run',
    text: () =>
      i18n._(
        msg`This mod wrote errors to the game's log the last time you played. It may be broken, out of date, or missing something it needs.`,
      ),
  },
  loadFailure: {
    anchor: 'failed-to-load',
    text: () =>
      i18n._(
        msg`The loader's log shows this plugin did not start, so its features are missing in the game. A wrong game version or a missing requirement is the usual cause.`,
      ),
  },
  setting: {
    anchor: 'setting-suggestion',
    text: () =>
      i18n._(msg`A mod works better, or only works, with a setting changed. Nothing is wrong yet.`),
  },
  damaged: {
    anchor: 'damaged-files',
    text: () =>
      i18n._(
        msg`Files of this stored mod went missing, changed or appeared after Mortar stored it. An antivirus, a disk fault or a manual edit can cause that, and the mod may misbehave.`,
      ),
  },
  pluginClash: {
    anchor: 'plugins-shipped-twice',
    text: () =>
      i18n._(
        msg`Two enabled packages carry the same plugin. The loader starts only one of them, so you may be running the older copy.`,
      ),
  },
  deprecated: {
    anchor: 'deprecated-packages',
    text: () =>
      i18n._(
        msg`The author marked this package as deprecated. It will not get fixes and may break with the next game update.`,
      ),
  },
  drift: {
    anchor: 'changed-outside-mortar',
    text: () =>
      i18n._(
        msg`Something changed this profile's mods folder without Mortar, by adding, removing or editing a file. Mortar flags it so the profile matches what you expect.`,
      ),
  },
}

export const problemGuideUrl = (kind: Row['kind']): string => `${GUIDE}#${problemWhy[kind].anchor}`
