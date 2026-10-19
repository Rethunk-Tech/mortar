import { useEffect, useState } from 'react'
import {
  Categories,
  DefaultExcludes,
  HasCompat,
  Search,
  SearchableSources,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { openPage } from '../mods/menu.ts'
import { download } from '../queue/actions.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { BrowsePage } from './BrowsePage.tsx'
import type { BrowseSearch, BrowseSource } from './browseTypes.ts'

const KIND_INSTALL = 'install'

async function categoryNames(game: string, source: string): Promise<string[]> {
  return (await Categories(game, source)) ?? []
}

async function defaultExcludes(game: string, source: string): Promise<string[]> {
  return (await DefaultExcludes(game, source)) ?? []
}

// Defined once: BrowsePage re-runs its query when this identity changes, and each run costs a request to every source.
const search: BrowseSearch = async ({
  game: nextGame,
  source,
  text,
  page,
  profileID: nextProfile,
  filter,
}) => {
  const result = await Search(nextGame, source, text, page, nextProfile, filter)
  return {
    total: result.total,
    items: result.items ?? [],
    pages: result.pages ?? 0,
    hidden: result.hidden ?? 0,
    failed: result.failed ?? [],
  }
}

// Module-level, so the result cards that take them keep the same props across renders.
const downloadNexus = (modID: string) => {
  download([{ kind: KIND_INSTALL, modId: Number(modID), latest: true }]).catch(reportUnexpected)
}
const addGitHub = (repo: string) => {
  download([{ kind: KIND_INSTALL, repo }]).catch(reportUnexpected)
}
const addPackage = (pkg: string) => {
  download([{ kind: KIND_INSTALL, package: pkg }]).catch(reportUnexpected)
}
const addDirect = (source: string, id: string, picture: string) => {
  download([{ kind: KIND_INSTALL, source, package: id, picture }]).catch(reportUnexpected)
}

function BrowseHost({ game, profileID }: { game: string; profileID: string }) {
  const premium = useNexus((state) => state.premium)
  const [sources, setSources] = useState<BrowseSource[]>([])
  const [hasCompat, setHasCompat] = useState(false)
  useEffect(() => {
    HasCompat(game).then(setHasCompat).catch(reportUnexpected)
  }, [game])
  useEffect(() => {
    SearchableSources(game)
      .then((ids) => setSources(ids ?? []))
      .catch(reportUnexpected)
  }, [game])
  return (
    <BrowsePage
      game={game}
      profileID={profileID}
      premium={premium}
      hasCompat={hasCompat}
      sources={sources}
      search={search}
      categories={categoryNames}
      defaultExcludes={defaultExcludes}
      openUrl={openPage}
      downloadNexus={downloadNexus}
      addGitHub={addGitHub}
      addPackage={addPackage}
      addDirect={addDirect}
    />
  )
}

export { BrowseHost }
