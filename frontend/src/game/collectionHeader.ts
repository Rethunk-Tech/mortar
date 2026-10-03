export interface CollectionLink {
  linked: boolean
  name: string
  revision: number
  latest: number
  url: string
}

export interface CollectionHeaderView {
  line: { name: string; revision: number; url: string } | null
  review: number | null
}

export function collectionHeader(status: CollectionLink | null | undefined): CollectionHeaderView {
  if (!status?.linked) {
    return { line: null, review: null }
  }
  return {
    line: { name: status.name, revision: status.revision, url: status.url },
    review: status.latest > status.revision ? status.latest : null,
  }
}
