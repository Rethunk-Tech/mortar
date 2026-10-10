import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import {
  HandoffPage,
  InstallHandoffDownload,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts'
import { OpenWeb } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import type { InstallResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { i18n } from '../i18n/index.ts'
import { askHandoff } from './handoffConfirm.ts'

export interface HandoffRef {
  /** The source the page is on: 'patreon' or 'itch'. */
  kind: string
  id: string
  url: string
  /** When the page was opened in the browser (ms); only files saved after this belong to it. */
  openedAt: number
}

const MINUTE_MS = 60_000
/** How long an opened page keeps claiming the next saved file. */
const TTL_MINUTES = 30
export const HANDOFF_TTL_MS = TTL_MINUTES * MINUTE_MS

// The Patreon post or itch.io page the player was sent to, until the file they save from it is added, they paste
// another link, or it expires. The Add-mod dialog may be closed by the time the file lands. Mortar fetches nothing
// from the site: the page opens in the browser and the Downloads offer does the rest.
export const useHandoff = create<{
  page: HandoffRef | null
  set: (page: HandoffRef | null) => void
}>((set) => ({ page: null, set: (page) => set({ page }) }))

/** The remembered page while it is still within its time-to-live, else null. */
export const liveHandoff = (page: HandoffRef | null, now = Date.now()) =>
  page && now - page.openedAt <= HANDOFF_TTL_MS ? page : null

const forget = () => useHandoff.getState().set(null)

// Opens the page a pasted Patreon post or itch.io game page address names and remembers it; false when the text is
// neither, which also drops any page remembered from an earlier paste.
export async function startHandoff(text: string): Promise<boolean> {
  const page = await HandoffPage(text).catch(() => null)
  if (!page) {
    forget()
    return false
  }
  await OpenWeb(page.url)
  useHandoff.getState().set({ ...page, openedAt: Date.now() })
  return true
}

const pageName = (page: HandoffRef) =>
  page.kind === 'patreon' ? i18n._(msg`Patreon post ${page.id}`) : page.id

// The install call for a file offered from Downloads: recorded under the remembered page when the file was saved
// after the page was opened and the page has not expired, else `install` as it is.
export function downloadInstaller(
  target: { game: string; profileId: string; file: string; mtime: number },
  install: (game: string, profileId: string, file: string) => Promise<InstallResult>,
  now = Date.now(),
): () => Promise<InstallResult> {
  const { game, profileId, file, mtime } = target
  const { page } = useHandoff.getState()
  const live = liveHandoff(page, now)
  if (page && !live) {
    forget()
  }
  if (!live || mtime < live.openedAt) {
    return () => install(game, profileId, file)
  }
  return async () => {
    if (!(await askHandoff(file, pageName(live)))) {
      return install(game, profileId, file)
    }
    const res = await InstallHandoffDownload(game, profileId, file, live.kind, live.id)
    forget()
    return res
  }
}
