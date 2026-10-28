import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import {
  ExtraFolderMods,
  InstallExtraFolderMod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { gameBusy } from '../launch/busy.ts'
import { useLaunch } from '../launch/store.ts'
import { useMods } from '../mods/store.ts'
import { willImport } from '../profiles/gameModsFormat.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

// Installs that a change asked for while the game ran, run once it stops, by the folder that changed.
const waiting = new Map<string, () => void>()
let stopWaiting: (() => void) | null = null

const busy = (game: string) => {
  const { starting, status } = useLaunch.getState()
  return starting || gameBusy(status, game)
}

function whenIdle(game: string, dir: string, run: () => void) {
  if (!busy(game)) {
    run()
    return
  }
  if (!waiting.has(dir)) {
    useToasts.getState().push({
      kind: 'info',
      title: i18n._(msg`Your build changed; it is added when the game stops`),
      body: dir,
    })
  }
  waiting.set(dir, run)
  stopWaiting ??= useLaunch.subscribe(() => {
    if (busy(game)) {
      return
    }
    stopWaiting?.()
    stopWaiting = null
    const runs = [...waiting.entries()]
    waiting.clear()
    for (const [d, r] of runs) {
      whenIdle(game, d, r)
    }
  })
}

// The profile the folder installs into: the one picked for it, else the open profile of the open game.
function targetOf(game: string, dir: string) {
  const picked = useSettings.getState().games?.[game]?.extraModsProfiles?.[dir]
  const { game: open, openId, profiles } = useProfiles.getState()
  if (picked) {
    // A profile deleted since it was picked installs nothing, rather than failing on every change in the folder.
    return open?.id === game && !profiles.some((p) => p.id === picked) ? '' : picked
  }
  return open?.id === game ? openId : ''
}

const inFolder = (folder: string, dir: string) =>
  folder === dir || folder.startsWith(`${dir}/`) || folder.startsWith(`${dir}\\`)

async function installFolder(game: string, dir: string) {
  const profileId = targetOf(game, dir)
  if (!profileId) {
    return
  }
  const listed = (await ExtraFolderMods(game)).mods ?? []
  const folders = listed
    .filter((m) => willImport(m.status) && m.folder && inFolder(m.folder, dir))
    .map((m) => m.folder ?? '')
  const added: string[] = []
  let profileName = ''
  for (const folder of folders) {
    try {
      const res = await InstallExtraFolderMod(game, profileId, folder)
      if (res.fomod || res.remap) {
        useToasts.getState().push({
          kind: 'warning',
          title: i18n._(
            msg`${{ name: folder }} needs a choice to install; drop it on the window instead`,
          ),
        })
        continue
      }
      if (res.profile && (res.added?.length ?? 0) > 0) {
        added.push(...(res.added ?? []))
        profileName = res.profile.name
        if (useProfiles.getState().openId === profileId) {
          useProfiles.getState().replace(res.profile)
        }
      }
    } catch (e) {
      toastError(i18n._(msg`Could not add ${{ name: folder }}`), e)
    }
  }
  if (added.length === 0) {
    return
  }
  if (useProfiles.getState().openId === profileId) {
    await useMods.getState().load()
  }
  useToasts.getState().push({
    kind: 'success',
    title: i18n._(msg`Installed ${{ mods: listNames(added) }} into ${{ profile: profileName }}`),
    body: dir,
  })
}

/** Installs an extra mods folder's mods now, as when its files change: after it is added or its profile changes. */
export function installExtraFolder(game: string, dir: string) {
  whenIdle(game, dir, () => {
    installFolder(game, dir).catch(reportUnexpected)
  })
}

let listening = false

/** Keeps profiles current with a mod author's builds: each extra mods folder's mods install into its profile each
 * time its files change. */
export function initAutoInstall() {
  if (listening || typeof Events.On !== 'function') {
    return
  }
  listening = true
  Events.On('library:extra-folder', (event) => {
    const { game, dir } = event.data
    installExtraFolder(game, dir)
  })
}
