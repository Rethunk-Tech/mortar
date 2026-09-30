import { useEffect } from 'react'
import { Updates } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { Get } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useLoader } from '../loader/store.ts'
import { maybeToastSmapi } from './smapiToast.ts'

function ignore() {
  return
}

export function useStartupChecks() {
  useEffect(() => {
    const run = async () => {
      const s = await Get()
      if (s.checkModUpdatesOnStart !== false) {
        const last = s.lastProfile ?? {}
        await Promise.all(
          Object.entries(last)
            .filter(([game, profile]) => game && profile)
            .map(([game, profile]) => Updates(game, profile ?? '').catch(ignore)),
        )
      }
      if (s.tellWhenSmapiOut !== false) {
        const game = s.lastGame || 'stardew'
        await useLoader.getState().check(game)
        maybeToastSmapi(game)
      }
    }
    run().catch(ignore)
  }, [])
}
