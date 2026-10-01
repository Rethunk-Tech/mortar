import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import { PickArchives } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  InstallArchive,
  RemoveEntry,
  RollBack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useFomod } from '../fomod/store.ts'
import { i18n } from '../i18n/index.ts'
import { useLaunch } from '../launch/store.ts'
import { isLocked } from '../mods/locked.ts'
import { useMods } from '../mods/store.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { changeStillLatest } from '../toasts/history.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { type MissingOffer, offersFor } from './missingDeps.ts'

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
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(msg`Could not undo the install`),
      body: errorMessage(e),
    })
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

export const useInstall = create<{
  pending: number
  offers: MissingOffer[]
  dismissOffer: () => void
  install: (paths: string[]) => Promise<void>
  pick: () => Promise<void>
}>((set, get) => ({
  pending: 0,
  offers: [],
  dismissOffer: () => set((s) => ({ offers: s.offers.slice(1) })),
  install: async (paths) => {
    const { game, openId, profiles } = useProfiles.getState()
    const profile = profiles.find((p) => p.id === openId)
    if (
      routeGame(useNav.getState().route) === null ||
      !game ||
      !profile ||
      isLocked(useLaunch.getState().status, openId, startingProfile())
    ) {
      return
    }
    const { push } = useToasts.getState()
    const dependentIds: string[] = []
    set((s) => ({ pending: s.pending + paths.length }))
    for (const path of paths) {
      try {
        await installOneArchive(game, profile, path, dependentIds)
      } catch (e) {
        push({
          kind: 'error',
          title: i18n._(msg`Could not add ${fileName(path)}`),
          body: errorMessage(e),
        })
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
      useToasts.getState().push({
        kind: 'error',
        title: i18n._(msg`Could not open the file dialog`),
        body: errorMessage(e),
      })
    }
  },
}))

export function considerMissing(dependentIds: readonly string[]) {
  const next = offersFor(dependentIds, useMods.getState().problems)
  if (next.length === 0) {
    return
  }
  useInstall.setState((s) => ({ offers: [...s.offers, ...next] }))
}

export { entryForNames, shouldConsiderMissing, undoArchiveInstall }
