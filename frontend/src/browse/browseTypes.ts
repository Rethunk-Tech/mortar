import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'

type BrowseItem = Item

interface BrowsePageResult {
  total: number
  items: Item[]
  // A search across every source sets pages (its largest source's page count) and names sources that did not answer.
  pages?: number
  failed?: string[]
  hidden?: number
}

interface BrowseSource {
  id: string
  name: string
  // Why the source cannot be searched now, such as a missing API key; empty when it can.
  unavailable?: string
}

interface BrowseQuery {
  game: string
  source: string
  text: string
  page: number
  profileID: string
  filter: BrowseFilter
}

interface BrowseFilter {
  include: string[]
  exclude: string[]
  sort: string
  installed: string
  obsolete: string
  broken: string
}

type BrowseSearch = (query: BrowseQuery) => Promise<BrowsePageResult>

interface BrowsePageProps {
  game: string
  profileID: string
  premium: boolean
  // Whether the game has a compatibility list, which the Broken row needs.
  hasCompat: boolean
  sources: BrowseSource[]
  search: BrowseSearch
  categories: (game: string, source: string) => Promise<string[]>
  openUrl: (url: string) => void
  downloadNexus: (modID: string) => void
  addGitHub: (repo: string) => void
  addPackage: (id: string) => void
  addDirect: (source: string, id: string) => void
}

export type { BrowseFilter, BrowseItem, BrowsePageProps, BrowseSearch }
