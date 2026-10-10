import { create } from 'zustand'
import {
  InstallItchDownload,
  ItchPage,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts'
import { OpenWeb } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import type { InstallResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

export interface ItchPageRef {
  id: string
  url: string
  /** When the page was opened in the browser (ms); only files saved after this belong to it. */
  openedAt: number
}

const MINUTE_MS = 60_000
/** How long an opened page keeps claiming the next saved file. */
const TTL_MINUTES = 30
export const ITCH_PAGE_TTL_MS = TTL_MINUTES * MINUTE_MS

// The itch.io page the player was sent to, until the file they save from it is added, they paste another link, or it
// expires. Mortar fetches nothing from itch.io this way: the page opens in the browser and the Downloads offer does
// the rest.
export const useItchPage = create<{
  page: ItchPageRef | null
  set: (page: ItchPageRef | null) => void
}>((set) => ({ page: null, set: (page) => set({ page }) }))

/** The remembered page while it is still within its time-to-live, else null. */
export const liveItchPage = (page: ItchPageRef | null, now = Date.now()) =>
  page && now - page.openedAt <= ITCH_PAGE_TTL_MS ? page : null

export const forgetItchPage = () => useItchPage.getState().set(null)

// Opens the page a pasted itch.io address names and remembers it; false when the text is not a game page address,
// which also drops any page remembered from an earlier paste.
export async function startItchPage(text: string): Promise<boolean> {
  const page = await ItchPage(text).catch(() => null)
  if (!page) {
    forgetItchPage()
    return false
  }
  await OpenWeb(page.url)
  useItchPage.getState().set({ id: page.id, url: page.url, openedAt: Date.now() })
  return true
}

// The install call for a file offered from Downloads: recorded under the remembered page when the file was saved after
// the page was opened and the page has not expired, else `install` as it is.
export function itchDownloadInstaller(
  target: { game: string; profileId: string; file: string; mtime: number },
  install: (game: string, profileId: string, file: string) => Promise<InstallResult>,
  now = Date.now(),
): () => Promise<InstallResult> {
  const { game, profileId, file, mtime } = target
  const { page } = useItchPage.getState()
  const live = liveItchPage(page, now)
  if (page && !live) {
    forgetItchPage()
  }
  if (!live || mtime < live.openedAt) {
    return () => install(game, profileId, file)
  }
  return async () => {
    const res = await InstallItchDownload(game, profileId, file, live.id)
    forgetItchPage()
    return res
  }
}
