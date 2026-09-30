import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import { PickArchives } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import { InstallArchive } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useLaunch } from '../launch/store.ts'
import { isLocked } from '../mods/locked.ts'
import { useMods } from '../mods/store.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const PATH_SEPARATOR = /[\\/]/

const fileName = (path: string) => path.split(PATH_SEPARATOR).pop() ?? path

export const useInstall = create<{
  pending: number
  install: (paths: string[]) => Promise<void>
  pick: () => Promise<void>
}>((set, get) => ({
  pending: 0,
  install: async (paths) => {
    const { game, openId, profiles } = useProfiles.getState()
    const profile = profiles.find((p) => p.id === openId)
    if (
      routeGame(useNav.getState().route) === null ||
      !game ||
      !profile ||
      isLocked(useLaunch.getState().status, openId)
    ) {
      return
    }
    const { push } = useToasts.getState()
    set((s) => ({ pending: s.pending + paths.length }))
    for (const path of paths) {
      try {
        const { profile: next, added, updated } = await InstallArchive(game.id, profile.id, path)
        useProfiles.getState().replace(next)
        const names = (added ?? []).join(', ')
        push({
          kind: 'success',
          title: updated
            ? i18n._(msg`Updated ${names} in ${profile.name}`)
            : i18n._(msg`Added ${names} to ${profile.name}`),
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
    await useMods.getState().load()
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
