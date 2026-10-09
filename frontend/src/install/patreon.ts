import { create } from 'zustand'
import {
  InstallPatreonDownload,
  PatreonPost,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts'
import { OpenWeb } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import type { InstallResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

export interface PatreonPostRef {
  id: string
  url: string
  /** When the post was opened in the browser (ms); only files saved after this belong to it. */
  openedAt: number
}

const MINUTE_MS = 60_000
/** How long an opened post keeps claiming the next saved file. */
const TTL_MINUTES = 30
export const PATREON_POST_TTL_MS = TTL_MINUTES * MINUTE_MS

// The Patreon post the player was sent to, until the file they save from it is added, they paste a non-Patreon link,
// or it expires. The Add-mod dialog may be closed by the time the file lands. Mortar fetches nothing from Patreon:
// the post opens in the browser and the Downloads offer does the rest.
export const usePatreonPost = create<{
  post: PatreonPostRef | null
  set: (post: PatreonPostRef | null) => void
}>((set) => ({ post: null, set: (post) => set({ post }) }))

const forgetPatreonPost = () => usePatreonPost.getState().set(null)

// Opens the post a pasted Patreon address names and remembers it; false when the text is not a Patreon post address,
// which also drops any post remembered from an earlier paste.
export async function startPatreonPost(text: string): Promise<boolean> {
  const post = await PatreonPost(text).catch(() => null)
  if (!post) {
    forgetPatreonPost()
    return false
  }
  await OpenWeb(post.url)
  usePatreonPost.getState().set({ id: post.id, url: post.url, openedAt: Date.now() })
  return true
}

// The install call for a file offered from Downloads: recorded under the remembered post when the file was saved
// after the post was opened and the post has not expired.
export function downloadInstaller(
  target: { game: string; profileId: string; file: string; mtime: number },
  install: (game: string, profileId: string, file: string) => Promise<InstallResult>,
  now = Date.now(),
): () => Promise<InstallResult> {
  const { game, profileId, file, mtime } = target
  const { post } = usePatreonPost.getState()
  const expired = post !== null && now - post.openedAt > PATREON_POST_TTL_MS
  if (expired) {
    forgetPatreonPost()
  }
  if (!post || expired || mtime < post.openedAt) {
    return () => install(game, profileId, file)
  }
  return async () => {
    const res = await InstallPatreonDownload(game, profileId, file, post.id)
    forgetPatreonPost()
    return res
  }
}
