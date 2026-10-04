import { Search } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { openPage } from '../mods/menu.ts'
import { download } from '../queue/actions.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { BrowsePage } from './BrowsePage.tsx'
import type { BrowseSearch } from './browseTypes.ts'

const KIND_INSTALL = 'install'

function BrowseHost({ game, profileID }: { game: string; profileID: string }) {
  const premium = useNexus((state) => state.premium)
  const search: BrowseSearch = async ({
    game: nextGame,
    source,
    text,
    page,
    profileID: nextProfile,
  }) => {
    const result = await Search(nextGame, source, text, page, nextProfile)
    return { total: result.total, items: result.items ?? [] }
  }
  return (
    <BrowsePage
      game={game}
      profileID={profileID}
      premium={premium}
      hasCurseForgeKey={false}
      search={search}
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
    />
  )
}

export { BrowseHost }
