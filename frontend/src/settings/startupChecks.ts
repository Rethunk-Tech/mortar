import { useLingui } from '@lingui/react/macro'
import { useEffect } from 'react'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import {
  Get,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { LastRunCrashed } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { loadGameStatus } from '../games/status.ts'
import { useLoader } from '../loader/store.ts'
import { loadUpdates } from '../mods/updates.ts'
import { settledLastProfile } from '../profiles/lastProfile.ts'
import { reportBug } from '../shell/reportBug.ts'
import { useToasts } from '../toasts/store.ts'
import { lastRunCrashToast } from './crashToast.ts'
import { maybeToastLoader } from './loaderToast.ts'

function ignore() {
  return
}

function shouldRunStartupCheck(on: boolean | null | undefined): boolean {
  return on !== false
}

export function useStartupChecks() {
  const { t } = useLingui()
  useEffect(() => {
    const run = async () => {
      const s = await Get()
      const crashed = await LastRunCrashed()
      const toast = lastRunCrashToast(
        crashed,
        {
          title: t`Mortar closed unexpectedly last time`,
          body: t`Report it so it can be fixed.`,
          action: t`Report a bug`,
        },
        () => reportBug(s.lastGame ?? ''),
      )
      if (toast) {
        useToasts.getState().push(toast)
      }
      if (shouldRunStartupCheck(s.checkModUpdatesOnStart)) {
        const last = s.lastProfile ?? {}
        await Promise.all(
          Object.entries(last)
            .filter(([game, profile]) => game && profile)
            .map(async ([game, profile]) => {
              // The remembered profile may have been deleted since, which the update check would reject.
              const id = settledLastProfile((await List(game)) ?? [], profile)
              if (id !== profile) {
                await SetLastProfile(game, id)
              }
              if (id) {
                await loadUpdates(game, id)
              }
            })
            .map((p) => p.catch(ignore)),
        )
      }
      if (shouldRunStartupCheck(s.tellWhenSmapiOut)) {
        const game = s.lastGame || (await loadGameStatus()).games.find((g) => g.available)?.id
        if (game) {
          await useLoader.getState().check(game)
          maybeToastLoader(game)
        }
      }
    }
    run().catch(ignore)
  }, [t])
}

export { shouldRunStartupCheck }
