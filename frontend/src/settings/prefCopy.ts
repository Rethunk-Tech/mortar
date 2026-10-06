import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'

interface PrefCopy {
  label: string
  description?: string
  placeholder?: string
  options?: { value: string; label: string; hint?: string }[]
}

function windowAndMods(i18n: I18n): Record<string, PrefCopy> {
  return {
    onPlay: {
      label: i18n._(msg`When you press Play`),
      description: i18n._(msg`What Mortar does when the game starts, then restore when it exits.`),
      options: [
        {
          value: 'stay',
          label: i18n._(msg`Stay open`),
          hint: i18n._(msg`Mortar stays on screen beside the game.`),
        },
        {
          value: 'minimise',
          label: i18n._(msg`Minimise`),
          hint: i18n._(msg`Mortar minimises while you play and comes back when the game exits.`),
        },
        {
          value: 'hide',
          label: i18n._(msg`Hide to tray`),
          hint: i18n._(msg`Mortar leaves the taskbar and waits in the tray until the game exits.`),
        },
      ],
    },
    startScreen: {
      label: i18n._(msg`Start screen`),
      description: i18n._(msg`What Mortar shows when it opens.`),
      options: [
        {
          value: 'last',
          label: i18n._(msg`Last opened profile`),
          hint: i18n._(msg`Pick up where you left off.`),
        },
        {
          value: 'gameselect',
          label: i18n._(msg`Game select`),
          hint: i18n._(msg`Choose a game and profile each time.`),
        },
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
    lanAutoAcceptPaired: {
      label: i18n._(msg`Auto-accept from paired computers`),
      description: i18n._(
        msg`Accept profiles sent from computers you paired, and copy their mod files`,
      ),
    },
    defaultModsView: {
      label: i18n._(msg`Default mods view`),
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
        msg`When you enable a mod, also enable its required mods already in the profile`,
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
        { value: 'group', label: i18n._(msg`Group`) },
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
    theme: {
      label: i18n._(msg`Theme`),
      description: i18n._(msg`Mortar's colours. Follow system matches your computer.`),
      options: [
        { value: 'dark', label: i18n._(msg`Dark`) },
        { value: 'light', label: i18n._(msg`Light`) },
        { value: 'system', label: i18n._(msg`Follow system`) },
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
      label: i18n._(msg`Back up saves before Play`),
      description: i18n._(msg`When Mortar makes a save backup before launching`),
      options: [
        {
          value: 'changed',
          label: i18n._(msg`When mods changed`),
          hint: i18n._(msg`Back up saves only if the profile's mods changed since the last run.`),
        },
        {
          value: 'always',
          label: i18n._(msg`Every Play`),
          hint: i18n._(msg`Back up saves every time the game starts.`),
        },
        {
          value: 'never',
          label: i18n._(msg`Never`),
          hint: i18n._(msg`Start the game without a backup.`),
        },
      ],
    },
    saveBackupsKept: {
      label: i18n._(msg`Save backups kept`),
      description: i18n._(
        msg`How many backups to keep of each kind: before Play, before a mod update, before a restore. Pinned backups are never deleted.`,
      ),
    },
    extraModsFolder: {
      label: i18n._(msg`Extra mods folder`),
      description: i18n._(msg`A folder of unpacked mods to add from the Add menu`),
    },
    showDotHiddenMods: {
      label: i18n._(msg`Show hidden mods`),
      description: i18n._(
        msg`List mods inside a folder whose name starts with a dot, which SMAPI skips`,
      ),
    },
    oldFilesOnUpdate: {
      label: i18n._(msg`Files an update no longer includes`),
      description: i18n._(msg`What happens to files the new version of a mod leaves out`),
      options: [
        {
          value: 'ask',
          label: i18n._(msg`Ask`),
          hint: i18n._(msg`Set them aside and ask whether to keep or delete them.`),
        },
        {
          value: 'delete',
          label: i18n._(msg`Delete`),
          hint: i18n._(msg`Delete them; rolling back still restores the old version.`),
        },
        {
          value: 'keep',
          label: i18n._(msg`Keep`),
          hint: i18n._(msg`Carry them into the new version's folder.`),
        },
      ],
    },
    saveBackupHours: {
      label: i18n._(msg`Scheduled save backups`),
      description: i18n._(
        msg`Hours between backups of saves that changed, while Mortar is open (24 is daily, 0 is Off). Waits until the game closes.`,
      ),
    },
    saveBackupKeep: { label: i18n._(msg`Scheduled backups kept per save`) },
    runsKept: {
      label: i18n._(msg`Run logs kept`),
      description: i18n._(msg`Logs of past runs stored per profile`),
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
      description: i18n._(msg`Keep the zip after installing`),
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
      label: i18n._(msg`Downloaded mods no profile uses`),
      description: i18n._(
        msg`Days to keep downloaded mods that no profile uses. 0 keeps them forever.`,
      ),
    },
    trashRetentionDays: {
      label: i18n._(msg`Recently deleted retention`),
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
    updateDigest: {
      label: i18n._(msg`Update digest notification`),
      description: i18n._(
        msg`How often background checks report new mod updates. The In Mortar and desktop switches choose where.`,
      ),
      options: [
        { value: 'off', label: i18n._(msg`Off`) },
        { value: 'each', label: i18n._(msg`After each check`) },
        { value: 'daily', label: i18n._(msg`At most once a day`) },
      ],
    },
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
    smapiPin: {
      label: i18n._(msg`SMAPI version`),
      description: i18n._(msg`Empty follows the latest release`),
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
      description: i18n._(
        msg`Where Mod Manager Download links install. Empty uses the last opened profile.`,
      ),
    },
    offerNewDownloads: {
      label: i18n._(msg`Offer new downloads`),
      description: i18n._(
        msg`Offer to add new mod archives that land in your Downloads folder or Mortar's download folder`,
      ),
    },
  }
}

function batchPrefs(i18n: I18n): Record<string, PrefCopy> {
  return {
    profileOrder: {
      label: i18n._(msg`Sidebar profile order`),
      description: i18n._(msg`How profiles are ordered in the sidebar`),
      options: [
        { value: 'manual', label: i18n._(msg`Manual`) },
        { value: 'name', label: i18n._(msg`Name`) },
        { value: 'lastPlayed', label: i18n._(msg`Last played`) },
      ],
    },
    backupLocation: {
      label: i18n._(msg`Save backup folder`),
      description: i18n._(msg`Where save backups are kept. Empty uses the default.`),
    },
    autoRetryDownloads: {
      label: i18n._(msg`Auto-retry failed downloads`),
      description: i18n._(msg`How many times to retry a failed download`),
      options: [
        { value: 'off', label: i18n._(msg`Off`) },
        { value: '1', label: i18n._(msg`1 time`) },
        { value: '3', label: i18n._(msg`3 times`) },
      ],
    },
    pauseDownloadsWhilePlaying: {
      label: i18n._(msg`Pause downloads while the game runs`),
      description: i18n._(msg`Hold the download queue until the game exits`),
    },
    sidebarBadges: {
      label: i18n._(msg`Sidebar badge counts`),
      description: i18n._(msg`What the sidebar shows on profile badges`),
      options: [
        { value: 'problemsAndUpdates', label: i18n._(msg`Problems and updates`) },
        { value: 'problems', label: i18n._(msg`Problems only`) },
        { value: 'off', label: i18n._(msg`Off`) },
      ],
    },
    conflictScanDepth: {
      label: i18n._(msg`Conflict scan depth`),
      description: i18n._(msg`How thoroughly Mortar analyses overlapping files`),
      options: [
        { value: 'full', label: i18n._(msg`Full`) },
        { value: 'skipImages', label: i18n._(msg`Skip image overlap analysis`) },
      ],
    },
    shareIncludeDisabledMods: {
      label: i18n._(msg`Share switched-off mods`),
      description: i18n._(msg`Default for Share and Export. You can change it per share.`),
    },
    shareIncludeFomodChoices: {
      label: i18n._(msg`Share FOMOD choices`),
      description: i18n._(msg`Default for Share and Export. You can change it per share.`),
    },
    shareIncludeNotes: {
      label: i18n._(msg`Share notes`),
      description: i18n._(msg`Default for Share and Export. You can change it per share.`),
    },
    shareIncludeConfigFiles: {
      label: i18n._(msg`Share config files`),
      description: i18n._(msg`Default for Share and Export. You can change it per share.`),
    },
    showAdultContent: {
      label: i18n._(msg`Show adult mods in browse`),
      description: i18n._(
        msg`Browse hides mods their site marks as adult unless this is on. Installed mods are never hidden.`,
      ),
    },
    verifyNexusMD5: {
      label: i18n._(msg`Verify downloads with Nexus MD5`),
      description: i18n._(msg`Check the file hash when Nexus provides one`),
    },
    launchAtLogin: {
      label: i18n._(msg`Launch Mortar at login`),
      description: i18n._(msg`Start Mortar when you sign in to this computer`),
    },
    startMinimised: {
      label: i18n._(msg`Start minimised to tray`),
      description: i18n._(msg`Open in the tray instead of showing the window`),
    },
    rememberWindow: {
      label: i18n._(msg`Remember window size and position`),
      description: i18n._(msg`Restore the last window bounds on startup`),
    },
    extensionConnection: {
      label: i18n._(msg`Browser extension connection`),
      description: i18n._(
        msg`Allow shows Nexus pages what Mortar has. Off hides that; download links still arrive`,
      ),
      options: [
        { value: 'allow', label: i18n._(msg`Allow`) },
        { value: 'off', label: i18n._(msg`Off`) },
      ],
    },
  }
}

export function prefCopy(i18n: I18n, key: string): PrefCopy {
  return (
    windowAndMods(i18n)[key] ??
    displayAndData(i18n)[key] ??
    logsAndNexus(i18n)[key] ??
    batchPrefs(i18n)[key] ?? { label: key }
  )
}
