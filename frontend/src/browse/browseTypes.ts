import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'

type BrowseItem = Item

interface BrowsePageResult {
  total: number
  items: Item[]
  // A search across every source sets pages (its largest source's page count) and names sources that did not answer.
  pages?: number
  failed?: string[]
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
  addPackage: (id: string) => void
}

export type { BrowseItem, BrowsePageProps, BrowseSearch }
