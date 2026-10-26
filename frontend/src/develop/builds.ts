import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type {
  BuildResult,
  PackResult,
  ProjectInfo,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Build,
  Inspect,
  Pack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { useSettings } from '../settings/store.ts'
import { toastError } from '../toasts/report.ts'

interface ProjectRun {
  // What runs now, if anything.
  running: 'build' | 'pack' | null
  build: BuildResult | null
  pack: PackResult | null
  // When the last build finished, in ms.
  builtAt: number
}

const idle: ProjectRun = { running: null, build: null, pack: null, builtAt: 0 }

// Each project's last build and pack, by folder; a build the source watch starts lands here too.
export const useDevRuns = create<{ byDir: Record<string, ProjectRun> }>(() => ({ byDir: {} }))

const patch = (dir: string, next: Partial<ProjectRun>) =>
  useDevRuns.setState((s) => ({
    byDir: { ...s.byDir, [dir]: { ...(s.byDir[dir] ?? idle), ...next } },
  }))

export const runOf = (byDir: Record<string, ProjectRun>, dir: string) => byDir[dir] ?? idle

/** Builds the project; its output then installs into the development profile through its extra mods folder. */
export async function buildProject(game: string, dir: string) {
  if (runOf(useDevRuns.getState().byDir, dir).running) {
    return
  }
  patch(dir, { running: 'build' })
  try {
    patch(dir, { build: await Build(game, dir), builtAt: Date.now() })
    await refreshInfo(game, dir)
  } catch (e) {
    toastError(i18n._(msg`Could not build the mod`), e)
  } finally {
    patch(dir, { running: null })
  }
}

/** Builds the project in Release and checks the package against Thunderstore's rules. */
export async function packProject(game: string, dir: string) {
  if (runOf(useDevRuns.getState().byDir, dir).running) {
    return
  }
  patch(dir, { running: 'pack' })
  try {
    patch(dir, { pack: await Pack(game, dir) })
  } catch (e) {
    toastError(i18n._(msg`Could not pack the mod`), e)
  } finally {
    patch(dir, { running: null })
  }
}

// What each project's files say, by folder: read on show, after a build, and when the window comes back.
export const useProjectInfo = create<{ byDir: Record<string, ProjectInfo> }>(() => ({ byDir: {} }))

/** Rereads the project's details from its files; a rename made outside Mortar is followed there. */
export async function refreshInfo(game: string, dir: string) {
  try {
    const info = await Inspect(game, dir)
    useProjectInfo.setState((s) => ({ byDir: { ...s.byDir, [dir]: info } }))
  } catch (e) {
    toastError(i18n._(msg`Could not read the mod project`), e)
  }
}

let listening = false

/** When a project's sources change, rebuilds it if build on save is on, else rereads it so its card says it changed. */
export function initDevWatch() {
  if (listening || typeof Events.On !== 'function') {
    return
  }
  listening = true
  Events.On('dev:source', (event) => {
    const game = String(event.data)
    for (const p of useSettings.getState().games?.[game]?.devProjects ?? []) {
      if (p.watch) {
        buildProject(game, p.dir).catch(() => undefined)
      } else {
        refreshInfo(game, p.dir).catch(() => undefined)
      }
    }
  })
}
