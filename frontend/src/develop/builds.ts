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
  BuildOrder,
  Inspect,
  Pack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { installExtraFolder } from '../downloads/autoInstall.ts'
import { i18n } from '../i18n/index.ts'
import { useSettings } from '../settings/store.ts'
import { errorKind } from '../toasts/errorKind.ts'
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

// A build the backend refused until the user agrees to run the project's repository code; the dialog resumes it.
export const useTrustAsk = create<{ ask: { game: string; dir: string; pack: boolean } | null }>(
  () => ({ ask: null }),
)

/**
 * Builds the project; its output then installs into the development profile through its extra mods folder. A project
 * not yet trusted asks first; a source-watch build (quiet) only marks the card changed.
 */
export async function buildProject(game: string, dir: string, quiet = false) {
  if (runOf(useDevRuns.getState().byDir, dir).running) {
    return
  }
  patch(dir, { running: 'build' })
  try {
    const build = await Build(game, dir)
    patch(dir, { build, builtAt: Date.now() })
    await refreshInfo(game, dir, quiet)
    // A build that changed nothing writes no file for the folder watch to see, so a mod removed from the profile comes
    // back only when asked here. A source-watch build's own writes already reach the watch.
    const output = useSettings
      .getState()
      .games?.[game]?.devProjects?.find((p) => p.dir === dir)?.output
    if (build.ok && !quiet && output) {
      installExtraFolder(game, output)
    }
  } catch (e) {
    if (errorKind(e) === 'unconfirmed') {
      if (quiet) {
        await refreshInfo(game, dir, true)
      } else {
        useTrustAsk.setState({ ask: { game, dir, pack: false } })
      }
      return
    }
    toastError(i18n._(msg`Could not build the mod`), e)
  } finally {
    patch(dir, { running: null })
  }
}

/**
 * Builds every project at once, except that one waits for the projects it depends on: a wave builds together, each
 * project compiling against what the waves before it wrote. A failed build reports itself; the next wave still starts.
 */
export async function buildAll(game: string, dirs: string[]) {
  for (const wave of (await BuildOrder(dirs)) ?? []) {
    await Promise.all((wave ?? []).map((dir) => buildProject(game, dir)))
  }
}

/** Builds the project in Release and checks the package against Thunderstore's rules; null when it could not. */
export async function packProject(game: string, dir: string): Promise<PackResult | null> {
  if (runOf(useDevRuns.getState().byDir, dir).running) {
    return null
  }
  patch(dir, { running: 'pack' })
  try {
    const pack = await Pack(game, dir)
    patch(dir, { pack })
    return pack
  } catch (e) {
    if (errorKind(e) === 'unconfirmed') {
      useTrustAsk.setState({ ask: { game, dir, pack: true } })
      return null
    }
    toastError(i18n._(msg`Could not pack the mod`), e)
    return null
  } finally {
    patch(dir, { running: null })
  }
}

// What each project's files say, by folder: read on show, after a build, and when the window comes back.
export const useProjectInfo = create<{ byDir: Record<string, ProjectInfo> }>(() => ({ byDir: {} }))

/** Rereads the project's details from its files; a rename made outside Mortar is followed there. A background read
 * (quiet) that fails stays silent, so a broken project does not toast on every window focus. */
export async function refreshInfo(game: string, dir: string, quiet = false) {
  try {
    const info = await Inspect(game, dir)
    useProjectInfo.setState((s) => ({ byDir: { ...s.byDir, [dir]: info } }))
  } catch (e) {
    if (!quiet) {
      toastError(i18n._(msg`Could not read the mod project`), e)
    }
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
    const { game, dir } = event.data
    const sep = dir.includes('\\') ? '\\' : '/'
    for (const p of useSettings.getState().games?.[game]?.devProjects ?? []) {
      if (dir !== p.dir && !dir.startsWith(p.dir + sep)) {
        continue
      }
      if (p.watch) {
        buildProject(game, p.dir, true).catch(() => undefined)
      } else {
        refreshInfo(game, p.dir, true).catch(() => undefined)
      }
    }
  })
}

/**
 * Builds each trusted code project that was never built or changed since its last build, so what leaves this computer
 * is current. Returns the name of the first project that did not build, or null when all did.
 */
export async function buildChanged(game: string, dirs: string[]): Promise<string | null> {
  const info = useProjectInfo.getState().byDir
  const stale = dirs.filter((dir) => {
    const built = Math.max(runOf(useDevRuns.getState().byDir, dir).builtAt, info[dir]?.builtAt ?? 0)
    return built === 0 || info[dir]?.stale === true
  })
  await buildAll(game, stale)
  const failed = stale.find((dir) => runOf(useDevRuns.getState().byDir, dir).build?.ok !== true)
  if (failed === undefined) {
    return null
  }
  return (
    useSettings.getState().games?.[game]?.devProjects?.find((p) => p.dir === failed)?.name ?? failed
  )
}
