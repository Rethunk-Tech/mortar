const TOUR_SEEN_ID = 'tour'

function seenList(seen: readonly string[] | null | undefined): readonly string[] {
  return seen ?? []
}

function tourSeen(seen: readonly string[] | null | undefined): boolean {
  return seenList(seen).includes(TOUR_SEEN_ID)
}

function tourShouldRun(seen: readonly string[] | null | undefined): boolean {
  return !tourSeen(seen)
}

function tourMarkSeen(seen: readonly string[] | null | undefined): string[] {
  const list = seenList(seen)
  if (tourSeen(seen)) {
    return [...list]
  }
  return [...list, TOUR_SEEN_ID]
}

function tourClearSeen(seen: readonly string[] | null | undefined): string[] {
  return seenList(seen).filter((id) => id !== TOUR_SEEN_ID)
}

export { tourClearSeen, tourMarkSeen, tourShouldRun }
