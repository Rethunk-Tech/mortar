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
}

// The Patreon post the player was sent to, until the file they save from it is added. Mortar fetches nothing from
// Patreon: the post opens in the browser and the Downloads offer does the rest.
export const usePatreonPost = create<{
  post: PatreonPostRef | null
  set: (post: PatreonPostRef | null) => void
}>((set) => ({ post: null, set: (post) => set({ post }) }))

// Opens the post a pasted Patreon address names and remembers it; false when the text is not a Patreon post address.
export async function startPatreonPost(text: string): Promise<boolean> {
  const post = await PatreonPost(text).catch(() => null)
  if (!post) {
    return false
  }
  await OpenWeb(post.url)
  usePatreonPost.getState().set({ id: post.id, url: post.url })
  return true
}

// The install call for a file offered from Downloads: recorded under the remembered post when there is one.
export function downloadInstaller(
  game: string,
  profileId: string,
  file: string,
  install: (game: string, profileId: string, file: string) => Promise<InstallResult>,
): () => Promise<InstallResult> {
  const post = usePatreonPost.getState().post
  if (!post) {
    return () => install(game, profileId, file)
  }
  return async () => {
    const res = await InstallPatreonDownload(game, profileId, file, post.id)
    usePatreonPost.getState().set(null)
    return res
  }
}
