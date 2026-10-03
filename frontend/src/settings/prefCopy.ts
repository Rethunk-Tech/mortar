import type { useLingui } from '@lingui/react/macro'

type T = ReturnType<typeof useLingui>['t']

interface PrefCopy {
  label: string
  description?: string
  placeholder?: string
  options?: { value: string; label: string }[]
}

function windowAndMods(t: T): Record<string, PrefCopy> {
  return {
    onPlay: {
      label: t`When you press Play`,
      description: t`What Mortar does when the game starts, then restore when it exits.`,
      options: [
        { value: 'stay', label: t`Stay open` },
        { value: 'minimise', label: t`Minimise` },
        { value: 'hide', label: t`Hide to tray` },
      ],
    },
    startScreen: {
      label: t`Start screen`,
      description: t`Where Mortar opens after the launcher check.`,
      options: [
        { value: 'last', label: t`Last opened profile` },
        { value: 'gameselect', label: t`Game Select` },
      ],
    },
    defaultLaunchMethod: {
      label: t`Default launch`,
      description: t`How Play starts the game. Direct skips Steam's overlay and playtime.`,
      options: [
        { value: 'steam', label: t`Steam` },
        { value: 'direct', label: t`Direct` },
      ],
    },
    showSmapiConsole: {
      label: t`SMAPI console window`,
      description: t`Show SMAPI's own window on a direct launch`,
    },
    lanName: {
      label: t`Device name`,
      description: t`How this Mortar appears to nearby installations`,
      placeholder: t`This computer`,
    },
    lanAutoAcceptSameAccount: {
      label: t`Auto-accept from this Nexus account`,
      description: t`Take LAN shares from machines signed in to the same Nexus account`,
    },
    defaultModsView: {
      label: t`Default Mods view`,
      description: t`Grid or list for new sessions`,
      options: [
        { value: 'grid', label: t`Grid` },
        { value: 'list', label: t`List` },
      ],
    },
    gridCardSize: {
      label: t`Grid card size`,
      options: [
        { value: 'small', label: t`Small` },
        { value: 'medium', label: t`Medium` },
        { value: 'large', label: t`Large` },
      ],
    },
    showAuthorOnCards: {
      label: t`Author on cards`,
      description: t`Show the author line on grid cards`,
    },
    enableRequirements: {
      label: t`Auto-enable requirements`,
      description: t`When you switch a mod on, also enable its required mods already in the profile`,
      options: [
        { value: 'always', label: t`Always` },
        { value: 'ask', label: t`Ask` },
        { value: 'never', label: t`Never` },
      ],
    },
    missingRequirements: {
      label: t`Missing requirements on install`,
      description: t`What to do when an installed mod still needs other mods`,
      options: [
        { value: 'ask', label: t`Ask` },
        { value: 'autodownload', label: t`Download them` },
        { value: 'never', label: t`Never` },
      ],
    },
    reuseFomodChoices: {
      label: t`Reuse FOMOD choices`,
      description: t`Skip the installer wizard when saved choices still fit`,
    },
    listGroupBy: {
      label: t`Default grouping`,
      options: [
        { value: 'none', label: t`None` },
        { value: 'status', label: t`Status` },
        { value: 'category', label: t`Category` },
        { value: 'source', label: t`Source` },
        { value: 'tag', label: t`Tag` },
        { value: 'framework', label: t`Framework` },
        { value: 'author', label: t`Author` },
      ],
    },
  }
}

function displayAndData(t: T): Record<string, PrefCopy> {
  return {
    listSortColumn: { label: t`Default sort` },
    listSortDir: { label: t`Default sort` },
    confirmRemovals: {
      label: t`Confirm removals`,
      description: t`Ask before removing mods from a profile`,
    },
    cosmeticConflicts: {
      label: t`Harmless conflicts`,
      description: t`Cosmetic overlaps on the Problems tab`,
      options: [
        { value: 'collapsed', label: t`Collapsed` },
        { value: 'expanded', label: t`Expanded` },
        { value: 'hidden', label: t`Hidden` },
      ],
    },
    backgroundBadgeChecks: {
      label: t`Badge checks for other profiles`,
      description: t`Check updates and problems in the background for profiles you are not looking at`,
    },
    dates: {
      label: t`Dates`,
      description: t`How timestamps are shown`,
      options: [
        { value: 'relative', label: t`Relative` },
        { value: 'absolute', label: t`Absolute` },
      ],
    },
    density: {
      label: t`Density`,
      description: t`Spacing of buttons and lists`,
      options: [
        { value: 'comfortable', label: t`Comfortable` },
        { value: 'compact', label: t`Compact` },
      ],
    },
    reduceMotion: {
      label: t`Reduce motion`,
      description: t`Shorter animations. Honour the OS unless you override it.`,
      options: [
        { value: 'system', label: t`Honour OS` },
        { value: 'always', label: t`Always` },
        { value: 'never', label: t`Never` },
      ],
    },
    profileHero: {
      label: t`Profile hero`,
      description: t`The banner at the top of a profile`,
      options: [
        { value: 'full', label: t`Full` },
        { value: 'compact', label: t`Compact` },
        { value: 'hidden', label: t`Hidden` },
      ],
    },
    notifyDownloadFinished: { label: t`Downloads finished` },
    notifyDownloadFailed: { label: t`Download failed` },
    notifyRunCrashed: { label: t`Run crashed` },
    backupBeforePlay: {
      label: t`Backup before Play`,
      description: t`When Mortar zips Saves before launching`,
      options: [
        { value: 'changed', label: t`When mods changed` },
        { value: 'always', label: t`Every Play` },
        { value: 'never', label: t`Never` },
      ],
    },
    launchBackupsKept: { label: t`Launch backups kept` },
    runsKept: { label: t`Run logs kept`, description: t`Stored SMAPI logs per profile` },
    consoleLogCap: { label: t`Console log cap`, description: t`Newest lines kept in the Console` },
  }
}

function logsAndNexus(t: T): Record<string, PrefCopy> {
  return {
    consoleLevel: {
      label: t`Console level`,
      description: t`Live log starts at this level and above`,
      options: [
        { value: 'trace', label: t`Trace` },
        { value: 'debug', label: t`Debug` },
        { value: 'info', label: t`Info` },
        { value: 'warn', label: t`Warn` },
        { value: 'error', label: t`Error` },
      ],
    },
    consoleTimestamps: { label: t`Console timestamps` },
    consoleFollow: { label: t`Follow live log` },
    keepDownloadArchives: {
      label: t`Keep downloaded archives`,
      description: t`Leave the zip after it is installed into the store`,
    },
    downloadFolder: {
      label: t`Download folder`,
      description: t`Archives land here instead of Mortar's downloads folder. Empty uses the default.`,
    },
    driftChecks: {
      label: t`Modified outside Mortar`,
      description: t`Scan the mods folder for changes Mortar did not make`,
    },
    storeRetentionDays: {
      label: t`Unused store items`,
      description: t`Days to keep unused store items. 0 keeps them forever.`,
    },
    trashRetentionDays: {
      label: t`Trash retention`,
      description: t`Days a deleted profile stays restorable`,
    },
    historyEventsKept: { label: t`History events kept`, description: t`Per profile` },
    autoInstallMortarUpdates: {
      label: t`Install Mortar updates automatically`,
      description: t`Download and stage a found update without asking`,
    },
    updateCheckIntervalMinutes: {
      label: t`Check interval`,
      description: t`Minutes between background update checks`,
    },
    notifyModUpdates: { label: t`Notify when updates are found` },
    updateModsBeforePlayDefault: {
      label: t`Update mods before Play on new profiles`,
      description: t`Default for a profile you just created`,
    },
    checkModUpdatesOnStart: { label: t`Check for mod updates when Mortar starts` },
    smapiBuilds: {
      label: t`SMAPI unofficial builds`,
      description: t`Pre-releases and unofficial SMAPI updates. Show lists them; Include lets Update all install them.`,
      options: [
        { value: 'never', label: t`Never` },
        { value: 'show', label: t`Show` },
        { value: 'include', label: t`Include` },
      ],
    },
    autoTrackNexus: {
      label: t`Auto-track installed mods`,
      description: t`Track a Nexus mod when Mortar installs it`,
    },
    parallelDownloads: {
      label: t`Parallel downloads`,
      description: t`Premium and GitHub downloads at once`,
    },
    nxmDefaultProfile: {
      label: t`Default profile for Nexus links`,
      description: t`Where nxm downloads go. Empty follows the last opened profile.`,
    },
  }
}

export function prefCopy(t: T, key: string): PrefCopy {
  return windowAndMods(t)[key] ?? displayAndData(t)[key] ?? logsAndNexus(t)[key] ?? { label: key }
}
