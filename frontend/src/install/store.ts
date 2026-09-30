import { t } from '@lingui/core/macro'
import { create } from 'zustand'
import { PickArchives } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import { InstallArchive } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useMods } from '../mods/store.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useToasts } from '../toasts/store.ts'

const message = (e: unknown) => (e instanceof Error ? e.message : String(e))

const fileName = (path: string) => path.split(/[\\/]/).pop() ?? path

export const useInstall = create<{
  pending: number
  install: (paths: string[]) => Promise<void>
  pick: () => Promise<void>
}>((set, get) => ({
  pending: 0,
  install: async (paths) => {
    const { game, openId, profiles } = useProfiles.getState()
    const profile = profiles.find((p) => p.id === openId)
    if (useNav.getState().route.name !== 'game' || !game || !profile) {
      return
    }
    const push = useToasts.getState().push
    set((s) => ({ pending: s.pending + paths.length }))
    for (const path of paths) {
      try {
        const { profile: next, added } = await InstallArchive(game.id, profile.id, path)
        useProfiles.getState().replace(next)
        const names = (added ?? []).join(', ')
        push({ kind: 'success', title: t`Added ${names} to ${profile.name}` })
      } catch (e) {
        push({ kind: 'error', title: t`Could not add ${fileName(path)}`, body: message(e) })
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
      useToasts
        .getState()
        .push({ kind: 'error', title: t`Could not open the file dialog`, body: message(e) })
    }
  },
}))
