import { useEffect, useState } from 'react'
import type { HostValue } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/models.ts'
import { useGameBusy } from '../../launch/store.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { configApi } from './api.ts'
import { useTypedConfig } from './store.ts'

const HOST_POLL_MS = 2000

// What a ConfigSync host sent for the open file's shared settings, by section/key, asked again while the editor is
// open while the game runs. Empty otherwise.
export function useHostValues(asks: boolean): Map<string, HostValue> {
  const target = useTypedConfig((s) => s.target)
  const file = useTypedConfig((s) => s.current)
  const running = useGameBusy(target?.game)
  const [values, setValues] = useState<Map<string, HostValue>>(() => new Map())
  useEffect(() => {
    setValues(new Map())
    if (!(asks && running && target && file)) {
      return
    }
    let live = true
    // The poll mostly finds what it found last time; only a change redraws the editor.
    let last = ''
    const ask = () => {
      configApi
        .hostValues(target, file)
        .then((rows) => {
          const seen = JSON.stringify(rows)
          if (live && seen !== last) {
            last = seen
            setValues(new Map(rows.map((r) => [`${r.section}/${r.key}`, r])))
          }
        })
        .catch((e: unknown) => {
          clearInterval(timer)
          reportUnexpected(e)
        })
    }
    const timer = setInterval(ask, HOST_POLL_MS)
    ask()
    return () => {
      live = false
      clearInterval(timer)
    }
  }, [asks, running, target, file])
  return values
}
