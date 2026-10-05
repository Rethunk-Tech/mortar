import { useEffect, useState } from 'react'
import {
  Categories,
  HasCompat,
  Search,
  SearchableSources,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { openPage } from '../mods/menu.ts'
import { download } from '../queue/actions.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { BrowsePage } from './BrowsePage.tsx'
import type { BrowseSearch } from './browseTypes.ts'

const KIND_INSTALL = 'install'

async function categoryNames(game: string, source: string): Promise<string[]> {
  return (await Categories(game, source)) ?? []
}

function BrowseHost({ game, profileID }: { game: string; profileID: string }) {
  const premium = useNexus((state) => state.premium)
  const [sources, setSources] = useState<{ id: string; name: string }[]>([])
  const [hasCompat, setHasCompat] = useState(false)
  useEffect(() => {
    HasCompat(game).then(setHasCompat).catch(reportUnexpected)
  }, [game])
  useEffect(() => {
    SearchableSources(game)
      .then((ids) => setSources(ids ?? []))
      .catch(reportUnexpected)
  }, [game])
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
  return (
    <BrowsePage
      game={game}
      profileID={profileID}
      premium={premium}
      hasCompat={hasCompat}
      sources={sources}
      search={search}
      categories={categoryNames}
      openUrl={(url) => {
        openPage(url).catch(reportUnexpected)
      }}
      downloadNexus={(modID) => {
        download([{ kind: KIND_INSTALL, modId: Number(modID), latest: true }]).catch(
          reportUnexpected,
        )
      }}
      addGitHub={(repo) => {
        download([{ kind: KIND_INSTALL, repo }]).catch(reportUnexpected)
      }}
      addPackage={(pkg) => {
        download([{ kind: KIND_INSTALL, package: pkg }]).catch(reportUnexpected)
      }}
      addDirect={(source, id) => {
        download([{ kind: KIND_INSTALL, source, package: id }]).catch(reportUnexpected)
      }}
    />
  )
}

export { BrowseHost }
