import { useEffect } from 'react'
import { Get } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useLoader } from '../loader/store.ts'
import { loadUpdates } from '../mods/updates.ts'
import { maybeToastSmapi } from './smapiToast.ts'

function ignore() {
  return
}

function shouldRunStartupCheck(on: boolean | null | undefined): boolean {
  return on !== false
}

export function useStartupChecks() {
  useEffect(() => {
    const run = async () => {
      const s = await Get()
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
  }, [])
}

export { shouldRunStartupCheck }
