import { useToasts } from '../toasts/store.ts'

const DOWNLOAD = 'https://mortar.rethunk.tech/download/'
const QUIET_MS = 4000

// What needs the internet, the player's own computer or an account, which a page cannot stand in for.
const OUTSIDE = new Set([
  'archivesvc.InstallDownload',
  'archivesvc.InstallHandoffDownload',
  'browse.Search',
  'configsvc.OpenText',
  'game.OpenScreenshotsFolder',
  'lan.Pair',
  'lan.PairCode',
  'loadersvc.Install',
  'loadersvc.InstallVersion',
  'nexussvc.SignIn',
  'nexussvc.StartSSO',
  'opener.OpenSteamValidate',
  'opener.OpenWeb',
  'picker.PickArchives',
  'picker.PickExecutable',
  'picker.PickFolder',
  'picker.PickImage',
  'picker.PickPackFile',
  'picker.ReadTextFile',
  'picker.SaveFile',
  'profile.InstallArchive',
  'profile.InstallBuildFolder',
  'profile.InstallExtraFolderMod',
  'profile.InstallFomod',
  'profile.InstallOverlay',
  'profile.OpenConfig',
  'profile.OpenConsolePath',
  'profile.OpenFolder',
  'profile.ShowFiles',
  'queue.Add',
  'queue.InstallAnyway',
  'queue.OpenPage',
  'queue.Retry',
  'savessvc.OpenBackupsFolder',
  'savessvc.OpenSaveFolder',
  'settings.ChooseBackgroundImage',
  'settings.ChooseGameFolder',
  'settings.OpenDataFolder',
  'sharesvc.ShowExportedCollection',
  'source/itch.SignIn',
  'support.ShowDiagnostics',
  'tools.Launch',
  'updatesvc.Install',
])

let shownAt = 0

/** Whether a call the recording lacks needs the real app; says so to the visitor, once in a while. */
export function outsideDemo(name: string): boolean {
  if (!OUTSIDE.has(name)) {
    return false
  }
  if (Date.now() - shownAt > QUIET_MS) {
    shownAt = Date.now()
    useToasts.getState().push({
      kind: 'info',
      title: 'This is a demo',
      body: 'Downloads, sign-ins and your own files need Mortar on your computer.',
      // The demo runs in the website's frame; the download page replaces the whole site.
      action: { label: 'Download Mortar', run: () => window.open(DOWNLOAD, '_top') },
    })
  }
  return true
}
