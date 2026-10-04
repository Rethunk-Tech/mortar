import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import { PickArchives } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type {
  Profile,
  RemapAsk,
  Source,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  InstallArchive,
  InstallRemap,
  RemoveEntry,
  RollBack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import {
  Add,
  AnswerRoot,
  FailRoot,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { useFomod } from '../fomod/store.ts'
import { i18n } from '../i18n/index.ts'
import { useLaunch } from '../launch/store.ts'
import { considerEnableRequirements } from '../mods/enableRequirementsApply.ts'
import { isLocked } from '../mods/locked.ts'
import { useMods } from '../mods/store.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { changeStillLatest } from '../toasts/history.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { type MissingOffer, offersFor, wantsOf } from './missingDeps.ts'

function dropInstallGate(hasRoute: boolean, hasTarget: boolean, locked: boolean) {
  if (!(hasRoute && hasTarget)) {
    return 'skip' as const
  }
  if (locked) {
    return 'locked' as const
  }
  return 'ok' as const
}

function startingProfile() {
  const { starting, startingProfile: id } = useLaunch.getState()
  return starting ? id : ''
}

const PATH_SEPARATOR = /[\\/]/

const fileName = (path: string) => path.split(PATH_SEPARATOR).pop() ?? path

function entryForNames(profile: Profile, names: string[]) {
  return (profile.entries ?? []).find((e) => (e.mods ?? []).some((m) => names.includes(m.name)))
}

async function undoArchiveInstall(
  game: string,
  profileId: string,
  entryKey: string,
  updated: boolean,
) {
  if (isLocked(useLaunch.getState().status, profileId, startingProfile())) {
    return
  }
  try {
    const next = updated
      ? await RollBack(game, profileId, entryKey)
      : await RemoveEntry(game, profileId, entryKey)
    useProfiles.getState().replace(next)
  } catch (e) {
    toastError(i18n._(msg`Could not undo the install`), e)
    return
  }
  await useMods.getState().load()
}

// Replacing a mod with the same version is not an update, so the toast says so.
function installTitle(
  mods: string,
  profileName: string,
  updated: boolean,
  versionChanged: boolean,
) {
  if (!updated) {
    return i18n._(msg`Added ${mods} to ${profileName}`)
  }
  return versionChanged
    ? i18n._(msg`Updated ${mods} in ${profileName}`)
    : i18n._(msg`Replaced ${mods} in ${profileName}`)
}

function shouldConsiderMissing(openId: string, installedProfileId: string): boolean {
  return openId === installedProfileId
}

async function maybeFinishInstall(profileId: string, dependentIds: string[]) {
  if (!shouldConsiderMissing(useProfiles.getState().openId, profileId)) {
    return
  }
  await useMods.getState().load()
  const { mods } = useMods.getState()
  const folded = new Set(dependentIds.map((id) => id.trim().toLowerCase()))
  await considerEnableRequirements(
    mods,
    mods.filter((m) => folded.has(m.uniqueId.trim().toLowerCase())),
    'install',
  )
  await useMods.getState().load()
  considerMissing(dependentIds)
}

async function installOneArchive(
  game: { id: string },
  profile: Profile,
  path: string,
  dependentIds: string[],
) {
  const { push } = useToasts.getState()
  const {
    profile: next,
    added,
    updated,
    versionChanged,
    fomod,
    remap,
  } = await InstallArchive(game.id, profile.id, path)
  if (fomod) {
    useFomod.getState().open({
      game: game.id,
      profileId: profile.id,
      key: fomod.key,
      source: fomod.source,
      ask: fomod,
    })
    return
  }
  if (remap) {
    useInstall.getState().openRemap({
      game: game.id,
      profileName: profile.name,
      profileId: profile.id,
      key: remap.key,
      source: remap.source,
      ask: remap,
    })
    return
  }
  useProfiles.getState().replace(next)
  const names = added ?? []
  const landed = entryForNames(next, names)
  for (const mod of landed?.mods ?? []) {
    if (mod.uniqueId) {
      dependentIds.push(mod.uniqueId)
    }
  }
  push({
    kind: 'success',
    title: installTitle(names.join(', '), profile.name, updated, versionChanged),
    picture: landed?.source.picture ?? '',
    ...(landed
      ? {
          action: {
            label: i18n._(msg`Undo`),
            run: () => undoArchiveInstall(game.id, profile.id, landed.key, updated),
            profileId: profile.id,
            live: () =>
              changeStillLatest(
                useProfiles.getState().profiles.find((p) => p.id === profile.id),
                landed.key,
                (landed.mods ?? []).map((m) => m.uniqueId),
              ),
          },
        }
      : {}),
  })
}

interface RemapSession {
  game: string
  profileId: string
  profileName: string
  key: string
  source: Source
  ask: RemapAsk
  queueId?: string
}

async function afterDroppedRemap(
  session: RemapSession,
  res: Awaited<ReturnType<typeof InstallRemap>>,
) {
  if (res.remap) {
    useInstall.setState({
      remap: { ...session, key: res.remap.key, source: res.remap.source, ask: res.remap },
    })
    return
  }
  if (res.fomod) {
    useFomod.getState().open({
      game: session.game,
      profileId: session.profileId,
      key: res.fomod.key,
      source: res.fomod.source,
      ask: res.fomod,
    })
    return
  }
  const next = res.profile
  useProfiles.getState().replace(next)
  const names = res.added ?? []
  const landed = entryForNames(next, names)
  useToasts.getState().push({
    kind: 'success',
    title: installTitle(
      names.join(', '),
      session.profileName,
      res.updated ?? false,
      res.versionChanged ?? false,
    ),
    picture: landed?.source.picture ?? '',
    ...(landed
      ? {
          action: {
            label: i18n._(msg`Undo`),
            run: () =>
              undoArchiveInstall(session.game, session.profileId, landed.key, res.updated ?? false),
            profileId: session.profileId,
            live: () =>
              changeStillLatest(
                useProfiles.getState().profiles.find((p) => p.id === session.profileId),
                landed.key,
                (landed.mods ?? []).map((m) => m.uniqueId),
              ),
          },
        }
      : {}),
  })
  const ids: string[] = []
  for (const mod of landed?.mods ?? []) {
    if (mod.uniqueId) {
      ids.push(mod.uniqueId)
    }
  }
  await maybeFinishInstall(session.profileId, ids)
}

export const useInstall = create<{
  pending: number
  offers: MissingOffer[]
  remap: RemapSession | null
  dismissOffer: () => void
  openRemap: (s: RemapSession) => void
  closeRemap: () => void
  chooseRoot: (root: string) => void
  install: (paths: string[]) => Promise<void>
  pick: () => Promise<void>
}>((set, get) => ({
  pending: 0,
  offers: [],
  remap: null,
  dismissOffer: () => set((s) => ({ offers: s.offers.slice(1) })),
  openRemap: (remap) => set({ remap }),
  closeRemap: () => {
    const session = get().remap
    if (session?.queueId) {
      FailRoot(session.queueId).catch(reportUnexpected)
    }
    set({ remap: null })
  },
  chooseRoot: (root) => {
    const session = get().remap
    if (!session) {
      return
    }
    set({ remap: null })
    if (session.queueId) {
      AnswerRoot(session.queueId, root).catch((e) => {
        toastError(i18n._(msg`Could not add the chosen folder`), e)
      })
      return
    }
    InstallRemap(session.game, session.profileId, session.key, root, session.source)
      .then((res) => afterDroppedRemap(session, res))
      .catch((e) => {
        toastError(i18n._(msg`Could not add the chosen folder`), e)
      })
  },
  install: async (paths) => {
    const { game, openId, profiles } = useProfiles.getState()
    const profile = profiles.find((p) => p.id === openId)
    const gate = dropInstallGate(
      routeGame(useNav.getState().route) !== null,
      Boolean(game && profile),
      isLocked(useLaunch.getState().status, openId, startingProfile()),
    )
    if (gate === 'skip') {
      return
    }
    if (gate === 'locked') {
      useToasts.getState().push({
        kind: 'warning',
        title: i18n._(msg`Stop the game to change mods.`),
      })
      return
    }
    if (!(game && profile)) {
      return
    }
    const dependentIds: string[] = []
    set((s) => ({ pending: s.pending + paths.length }))
    for (const path of paths) {
      try {
        await installOneArchive(game, profile, path, dependentIds)
      } catch (e) {
        toastError(i18n._(msg`Could not add ${fileName(path)}`), e)
      } finally {
        set((s) => ({ pending: s.pending - 1 }))
      }
    }
    await maybeFinishInstall(profile.id, dependentIds)
  },
  pick: async () => {
    try {
      const paths = (await PickArchives()) ?? []
      if (paths.length > 0) {
        await get().install(paths)
      }
    } catch (e) {
      toastError(i18n._(msg`Could not open the file dialog`), e)
    }
  },
}))

// A CLI install that needs a choice arrives here, so the window asks it exactly as for a dropped archive.
export function initInstallAsks() {
  Events.On('install:ask', (event) => {
    const { game, profile: profileId, fomod, remap } = event.data
    if (fomod) {
      useFomod
        .getState()
        .open({ game, profileId, key: fomod.key, source: fomod.source, ask: fomod })
      return
    }
    if (remap) {
      const profileName =
        useProfiles.getState().profiles.find((p) => p.id === profileId)?.name ?? ''
      useInstall.getState().openRemap({
        game,
        profileName,
        profileId,
        key: remap.key,
        source: remap.source,
        ask: remap,
      })
    }
  })
}

export function considerMissing(dependentIds: readonly string[]) {
  const mode = gamePrefs(useSettings.getState()).missingRequirements || 'ask'
  if (mode === 'never') {
    return
  }
  const next = offersFor(dependentIds, useMods.getState().problems)
  if (next.length === 0) {
    return
  }
  if (mode === 'autodownload') {
    const wants = next.flatMap((o) => wantsOf(o.missing))
    const { game, openId } = useProfiles.getState()
    if (wants.length > 0 && game && openId) {
      Add(
        wants.map((r) => ({
          kind: r.kind,
          modId: r.modId ?? 0,
          repo: r.repo ?? '',
          tag: r.tag ?? '',
          asset: r.asset ?? '',
          fileId: r.fileId ?? 0,
          name: r.name ?? '',
          fileName: r.fileName ?? '',
          version: r.version ?? '',
          currentKey: r.currentKey ?? '',
          batchId: '',
          latest: r.latest ?? false,
          game: game.id,
          profileId: openId,
        })),
      ).catch(reportUnexpected)
    }
    return
  }
  useInstall.setState((s) => ({ offers: [...s.offers, ...next] }))
}

export type { RemapSession }
export { dropInstallGate, entryForNames, shouldConsiderMissing, undoArchiveInstall }
