import { useLingui } from '@lingui/react/macro'
import { useEffect } from 'react'
import { Get } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { LastRunCrashed } from '../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { useLoader } from '../loader/store.ts'
import { loadUpdates } from '../mods/updates.ts'
import { reportBug } from '../shell/reportBug.ts'
import { useToasts } from '../toasts/store.ts'
import { lastRunCrashToast } from './crashToast.ts'
import { maybeToastSmapi } from './smapiToast.ts'

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
            .map(([game, profile]) => loadUpdates(game, profile ?? '').catch(ignore)),
        )
      }
      if (shouldRunStartupCheck(s.tellWhenSmapiOut)) {
        const game = s.lastGame || 'stardew'
        await useLoader.getState().check(game)
        maybeToastSmapi(game)
      }
    }
    run().catch(ignore)
  }, [t])
}

export { shouldRunStartupCheck }
