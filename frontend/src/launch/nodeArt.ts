// The picture a launch node draws after `failed` of its sources did not load, and whether the node's fallback mark
// (the game's initial) sits over it: a cover crop of a wide hero shows only a bland strip of sky, so it needs the mark
// to say which game it is.
export function nodeArt(
  srcs: string[],
  failed: number,
  marksLaterSrc: boolean,
): { src: string | undefined; mark: boolean } {
  const live = srcs.filter(Boolean)
  return { src: live[failed], mark: marksLaterSrc && failed > 0 && live[failed] !== undefined }
}
