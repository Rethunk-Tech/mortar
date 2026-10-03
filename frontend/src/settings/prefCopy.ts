import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'

interface PrefCopy {
  label: string
  description?: string
  placeholder?: string
  options?: { value: string; label: string }[]
}

function windowAndMods(i18n: I18n): Record<string, PrefCopy> {
  return {
    onPlay: {
      label: i18n._(msg`When you press Play`),
      description: i18n._(msg`What Mortar does when the game starts, then restore when it exits.`),
      options: [
        { value: 'stay', label: i18n._(msg`Stay open`) },
        { value: 'minimise', label: i18n._(msg`Minimise`) },
        { value: 'hide', label: i18n._(msg`Hide to tray`) },
      ],
    },
    startScreen: {
      label: i18n._(msg`Start screen`),
      description: i18n._(msg`Where Mortar opens after the launcher check.`),
      options: [
        { value: 'last', label: i18n._(msg`Last opened profile`) },
        { value: 'gameselect', label: i18n._(msg`Game Select`) },
      ],
    },
    defaultLaunchMethod: {
      label: i18n._(msg`Default launch`),
      description: i18n._(
        msg`How Play starts the game. Direct skips Steam's overlay and playtime.`,
      ),
      options: [
        { value: 'steam', label: i18n._(msg`Steam`) },
        { value: 'direct', label: i18n._(msg`Direct`) },
      ],
    },
    showSmapiConsole: {
      label: i18n._(msg`SMAPI console window`),
      description: i18n._(msg`Show SMAPI's own window on a direct launch`),
    },
    lanName: {
      label: i18n._(msg`Device name`),
      description: i18n._(msg`How this Mortar appears to nearby installations`),
      placeholder: i18n._(msg`This computer`),
    },
    lanAutoAcceptSameAccount: {
      label: i18n._(msg`Auto-accept from this Nexus account`),
      description: i18n._(msg`Take LAN shares from machines signed in to the same Nexus account`),
    },
    defaultModsView: {
      label: i18n._(msg`Default Mods view`),
      description: i18n._(msg`Grid or list for new sessions`),
      options: [
        { value: 'grid', label: i18n._(msg`Grid`) },
        { value: 'list', label: i18n._(msg`List`) },
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
    enableRequirements: {
      label: i18n._(msg`Auto-enable requirements`),
      description: i18n._(
        msg`When you switch a mod on, also enable its required mods already in the profile`,
      ),
      options: [
        { value: 'always', label: i18n._(msg`Always`) },
        { value: 'ask', label: i18n._(msg`Ask`) },
        { value: 'never', label: i18n._(msg`Never`) },
      ],
    },
    missingRequirements: {
      label: i18n._(msg`Missing requirements on install`),
      description: i18n._(msg`What to do when an installed mod still needs other mods`),
      options: [
        { value: 'ask', label: i18n._(msg`Ask`) },
        { value: 'autodownload', label: i18n._(msg`Download them`) },
        { value: 'never', label: i18n._(msg`Never`) },
      ],
    },
    reuseFomodChoices: {
      label: i18n._(msg`Reuse FOMOD choices`),
      description: i18n._(msg`Skip the installer wizard when saved choices still fit`),
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
        { value: 'author', label: i18n._(msg`Author`) },
      ],
    },
  }
}

function displayAndData(i18n: I18n): Record<string, PrefCopy> {
  return {
    listSortColumn: { label: i18n._(msg`Default sort`) },
    listSortDir: { label: i18n._(msg`Default sort`) },
    confirmRemovals: {
      label: i18n._(msg`Confirm removals`),
      description: i18n._(msg`Ask before removing mods from a profile`),
    },
    cosmeticConflicts: {
      label: i18n._(msg`Harmless conflicts`),
      description: i18n._(msg`Cosmetic overlaps on the Problems tab`),
      options: [
        { value: 'collapsed', label: i18n._(msg`Collapsed`) },
        { value: 'expanded', label: i18n._(msg`Expanded`) },
        { value: 'hidden', label: i18n._(msg`Hidden`) },
      ],
    },
    backgroundBadgeChecks: {
      label: i18n._(msg`Badge checks for other profiles`),
      description: i18n._(
        msg`Check updates and problems in the background for profiles you are not looking at`,
      ),
    },
    dates: {
      label: i18n._(msg`Dates`),
      description: i18n._(msg`How timestamps are shown`),
      options: [
        { value: 'relative', label: i18n._(msg`Relative`) },
        { value: 'absolute', label: i18n._(msg`Absolute`) },
      ],
    },
    density: {
      label: i18n._(msg`Density`),
      description: i18n._(msg`Spacing of buttons and lists`),
      options: [
        { value: 'comfortable', label: i18n._(msg`Comfortable`) },
        { value: 'compact', label: i18n._(msg`Compact`) },
      ],
    },
    reduceMotion: {
      label: i18n._(msg`Reduce motion`),
      description: i18n._(msg`Shorter animations. Honour the OS unless you override it.`),
      options: [
        { value: 'system', label: i18n._(msg`Honour OS`) },
        { value: 'always', label: i18n._(msg`Always`) },
        { value: 'never', label: i18n._(msg`Never`) },
      ],
    },
    profileHero: {
      label: i18n._(msg`Profile hero`),
      description: i18n._(msg`The banner at the top of a profile`),
      options: [
        { value: 'full', label: i18n._(msg`Full`) },
        { value: 'compact', label: i18n._(msg`Compact`) },
        { value: 'hidden', label: i18n._(msg`Hidden`) },
      ],
    },
    notifyDownloadFinished: { label: i18n._(msg`Downloads finished`) },
    notifyDownloadFailed: { label: i18n._(msg`Download failed`) },
    notifyRunCrashed: { label: i18n._(msg`Run crashed`) },
    backupBeforePlay: {
      label: i18n._(msg`Backup before Play`),
      description: i18n._(msg`When Mortar zips Saves before launching`),
      options: [
        { value: 'changed', label: i18n._(msg`When mods changed`) },
        { value: 'always', label: i18n._(msg`Every Play`) },
        { value: 'never', label: i18n._(msg`Never`) },
      ],
    },
    launchBackupsKept: { label: i18n._(msg`Launch backups kept`) },
    runsKept: {
      label: i18n._(msg`Run logs kept`),
      description: i18n._(msg`Stored SMAPI logs per profile`),
    },
    consoleLogCap: {
      label: i18n._(msg`Console log cap`),
      description: i18n._(msg`Newest lines kept in the Console`),
    },
  }
}

function logsAndNexus(i18n: I18n): Record<string, PrefCopy> {
  return {
    consoleLevel: {
      label: i18n._(msg`Console level`),
      description: i18n._(msg`Live log starts at this level and above`),
      options: [
        { value: 'trace', label: i18n._(msg`Trace`) },
        { value: 'debug', label: i18n._(msg`Debug`) },
        { value: 'info', label: i18n._(msg`Info`) },
        { value: 'warn', label: i18n._(msg`Warn`) },
        { value: 'error', label: i18n._(msg`Error`) },
      ],
    },
    consoleTimestamps: { label: i18n._(msg`Console timestamps`) },
    consoleFollow: { label: i18n._(msg`Follow live log`) },
    keepDownloadArchives: {
      label: i18n._(msg`Keep downloaded archives`),
      description: i18n._(msg`Leave the zip after it is installed into the store`),
    },
    downloadFolder: {
      label: i18n._(msg`Download folder`),
      description: i18n._(
        msg`Archives land here instead of Mortar's downloads folder. Empty uses the default.`,
      ),
    },
    driftChecks: {
      label: i18n._(msg`Modified outside Mortar`),
      description: i18n._(msg`Scan the mods folder for changes Mortar did not make`),
    },
    storeRetentionDays: {
      label: i18n._(msg`Unused store items`),
      description: i18n._(msg`Days to keep unused store items. 0 keeps them forever.`),
    },
    trashRetentionDays: {
      label: i18n._(msg`Trash retention`),
      description: i18n._(msg`Days a deleted profile stays restorable`),
    },
    historyEventsKept: {
      label: i18n._(msg`History events kept`),
      description: i18n._(msg`Per profile`),
    },
    autoInstallMortarUpdates: {
      label: i18n._(msg`Install Mortar updates automatically`),
      description: i18n._(msg`Download and stage a found update without asking`),
    },
    updateCheckIntervalMinutes: {
      label: i18n._(msg`Check interval`),
      description: i18n._(msg`Minutes between background update checks`),
    },
    notifyModUpdates: { label: i18n._(msg`Notify when updates are found`) },
    updateModsBeforePlayDefault: {
      label: i18n._(msg`Update mods before Play on new profiles`),
      description: i18n._(msg`Default for a profile you just created`),
    },
    checkModUpdatesOnStart: { label: i18n._(msg`Check for mod updates when Mortar starts`) },
    smapiBuilds: {
      label: i18n._(msg`SMAPI unofficial builds`),
      description: i18n._(
        msg`Pre-releases and unofficial SMAPI updates. Show lists them; Include lets Update all install them.`,
      ),
      options: [
        { value: 'never', label: i18n._(msg`Never`) },
        { value: 'show', label: i18n._(msg`Show`) },
        { value: 'include', label: i18n._(msg`Include`) },
      ],
    },
    autoTrackNexus: {
      label: i18n._(msg`Auto-track installed mods`),
      description: i18n._(msg`Track a Nexus mod when Mortar installs it`),
    },
    parallelDownloads: {
      label: i18n._(msg`Parallel downloads`),
      description: i18n._(msg`Premium and GitHub downloads at once`),
    },
    nxmDefaultProfile: {
      label: i18n._(msg`Default profile for Nexus links`),
      description: i18n._(msg`Where nxm downloads go. Empty follows the last opened profile.`),
    },
  }
}

export function prefCopy(i18n: I18n, key: string): PrefCopy {
  return (
    windowAndMods(i18n)[key] ??
    displayAndData(i18n)[key] ??
    logsAndNexus(i18n)[key] ?? { label: key }
  )
}
