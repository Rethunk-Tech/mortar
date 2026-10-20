import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { i18n } from '../i18n/index.ts'
import { useInstall } from '../install/store.ts'
import { gameBusy } from '../launch/busy.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

interface AutoInstall {
  game: string
  dir: string
}

// Folders that changed while the game ran, installed once it stops.
const waiting = new Set<string>()
let stopWaiting: (() => void) | null = null

const busy = (game: string) => {
  const { starting, status } = useLaunch.getState()
  return starting || gameBusy(status, game)
}

function install(game: string, dir: string) {
  if (useProfiles.getState().game?.id !== game) {
    return
  }
  if (!busy(game)) {
    useInstall.getState().installFromBuildFolder(dir).catch(reportUnexpected)
    return
  }
  if (!waiting.has(dir)) {
    useToasts.getState().push({
      kind: 'info',
      title: i18n._(msg`Your build changed; it is added when the game stops`),
      body: dir,
    })
  }
  waiting.add(dir)
  stopWaiting ??= useLaunch.subscribe(() => {
    if (busy(game)) {
      return
    }
    stopWaiting?.()
    stopWaiting = null
    const dirs = [...waiting]
    waiting.clear()
    for (const d of dirs) {
      install(game, d)
    }
  })
}

let listening = false

/** Installs a folder set to auto-install into the open profile each time its contents change, as a mod author's build. */
export function initAutoInstall() {
  if (listening || typeof Events.On !== 'function') {
    return
  }
  listening = true
  Events.On('library:auto-install', (event) => {
    const { game, dir } = event.data as AutoInstall
    install(game, dir)
  })
}
