import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/browse/models.ts'

type BrowseItem = Item

interface BrowsePageResult {
  total: number
  items: Item[]
}

interface BrowseQuery {
  game: string
  source: string
  text: string
  page: number
  profileID: string
}

type BrowseSearch = (query: BrowseQuery) => Promise<BrowsePageResult>

interface BrowsePageProps {
  game: string
  profileID: string
  premium: boolean
  hasCurseForgeKey: boolean
  search: BrowseSearch
  openUrl: (url: string) => void
  downloadNexus: (modID: string) => void
  addGitHub: (repo: string) => void
}

export type { BrowseItem, BrowsePageProps, BrowsePageResult, BrowseQuery, BrowseSearch }
