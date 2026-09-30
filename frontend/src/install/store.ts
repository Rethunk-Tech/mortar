import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import { PickArchives } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  InstallArchive,
  RemoveEntry,
  RollBack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
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
        const {
          profile: next,
          added,
          updated,
          versionChanged,
        } = await InstallArchive(game.id, profile.id, path)
        useProfiles.getState().replace(next)
        const names = added ?? []
        const entry = entryForNames(next, names)
        for (const mod of entry?.mods ?? []) {
          if (mod.uniqueId) {
            dependentIds.push(mod.uniqueId)
          }
        }
        let title: string
        if (!updated) {
          title = i18n._(msg`Added ${names.join(', ')} to ${profile.name}`)
        } else if (versionChanged) {
          title = i18n._(msg`Updated ${names.join(', ')} in ${profile.name}`)
        } else {
          title = i18n._(msg`Replaced ${names.join(', ')} in ${profile.name}`)
        }
        push({
          kind: 'success',
          title,
          picture: entry?.source.picture ?? '',
          ...(entry
            ? {
                action: {
                  label: i18n._(msg`Undo`),
                  run: () => undoArchiveInstall(game.id, profile.id, entry.key, updated),
                  profileId: profile.id,
                  live: () =>
                    changeStillLatest(
                      useProfiles.getState().profiles.find((p) => p.id === profile.id),
                      entry.key,
                      (entry.mods ?? []).map((m) => m.uniqueId),
                    ),
                },
              }
            : {}),
        })
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
