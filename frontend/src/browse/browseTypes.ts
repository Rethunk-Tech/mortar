import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'

type BrowseItem = Item

interface BrowsePageResult {
  total: number
  items: Item[]
}

interface BrowseSource {
  id: string
  name: string
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
  sources: BrowseSource[]
  search: BrowseSearch
  openUrl: (url: string) => void
  downloadNexus: (modID: string) => void
  addGitHub: (repo: string) => void
}

export type { BrowseItem, BrowsePageProps, BrowseSearch }
