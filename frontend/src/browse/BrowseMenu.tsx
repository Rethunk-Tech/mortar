import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { ExternalLink, EyeOff, Info, Link2, PackagePlus, Plus } from 'lucide-react'
import { memo, useState } from 'react'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import {
  HideBrowseAuthor,
  HideBrowseMod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { addDependency, type DependencyAsk, useDependencyAsk } from '../develop/dependencyAsk.ts'
import { useDevKinds } from '../develop/kinds.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useProfiles } from '../profiles/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { useSettings } from '../settings/store.ts'
import { copyText } from '../share/copyText.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { useOfflineReason } from '../shell/offlineText.ts'
import { reportError } from '../toasts/report.ts'
import { NEXUS, THUNDERSTORE } from './browseConstants.ts'
import type { BrowseItem } from './browseTypes.ts'
import { useBrowseSelection } from './selection.ts'
import { type ActionProps, useCardAction } from './useCardAction.tsx'
import { useBrowseView } from './view.ts'

// A mod of files has no optional dependency, so there is nothing to ask.
function askDependency(
  game: string,
  p: DevProject,
  item: BrowseItem,
  install: DependencyAsk['install'],
) {
  const ask = {
    game,
    dir: p.dir,
    project: p.name,
    pkg: item.id,
    name: item.name,
    version: item.version,
    install,
  }
  if (useDevKinds.getState().byDir[p.dir]) {
    addDependency(ask, false)
  } else {
    useDependencyAsk.setState({ ask })
  }
}

function Items({
  item,
  source,
  card,
  sourceNames,
  close,
  cardAction,
}: {
  item: BrowseItem
  source: string
  card: ActionProps
  sourceNames: Map<string, string>
  close: () => void
  cardAction: Pick<ReturnType<typeof useCardAction>, 'shownItem' | 'url' | 'installed' | 'add'>
}) {
  const { t } = useLingui()
  const game = useCurrentGame()
  const { shownItem, url, installed, add } = cardAction
  const openId = useProfiles((s) => s.openId)
  const projects = (useSettings((s) => s.games?.[game]?.devProjects) ?? []).filter(
    (p) => p.profile === openId,
  )
  // A development profile's mod declares what it needs; the player says whether it needs it or only uses it.
  const depend = (p: DevProject) => askDependency(game, p, shownItem, installed ? null : add)
  const choices = [{ source: item.source, url: item.url }, ...(item.alts ?? [])]
  const run = (fn: () => void) => () => {
    close()
    fn()
  }
  const opensFiles = source === NEXUS && !card.premium
  const signedIn = useNexus((state) => state.signedIn)
  const offline = useOfflineReason([source])
  let addWhy: string | undefined
  if (installed) {
    addWhy = t`Already in this profile.`
  } else if (shownItem.bundled) {
    addWhy = t`Mortar installs this itself.`
  } else if (source === NEXUS && !signedIn) {
    addWhy = t`Sign in to download`
  } else if (offline !== '') {
    addWhy = offline
  }
  return [
    <MenuAction
      key="details"
      icon={<Info size={16} />}
      label={t`Show details`}
      onClick={run(() => useBrowseSelection.getState().select({ item, source }))}
    />,
    shownItem.external === true ? (
      <MenuAction
        key="add"
        icon={<ExternalLink size={16} />}
        label={t`Open page`}
        onClick={run(() => card.openUrl(url))}
      />
    ) : (
      <MenuAction
        key="add"
        icon={opensFiles ? <ExternalLink size={16} /> : <Plus size={16} />}
        label={opensFiles ? t`Open files page` : t`Add`}
        disabled={addWhy !== undefined}
        reason={addWhy}
        onClick={run(add)}
      />
    ),
    ...(source === THUNDERSTORE && shownItem.version
      ? projects.map((p) => (
          <MenuAction
            key={`depend-${p.dir}`}
            icon={<PackagePlus size={16} />}
            label={t`Add as dependency of ${{ name: p.name }}…`}
            onClick={run(() => depend(p))}
          />
        ))
      : []),
    // An external item's own page is the Open page above; only the other sources' pages are listed.
    ...choices
      .filter((c) => !(shownItem.external === true && c.url === url))
      .map((c) => (
        <MenuAction
          key={`open-${c.source}`}
          icon={<ExternalLink size={16} />}
          label={t`Open page on ${{ source: sourceNames.get(c.source) ?? c.source }}`}
          onClick={run(() => card.openUrl(c.url))}
        />
      )),
    <MenuAction
      key="copy"
      icon={<Link2 size={16} />}
      label={t`Copy link`}
      onClick={run(() => {
        copyText(url, t`Link copied`)
      })}
    />,
    <MenuAction
      key="hide"
      icon={<EyeOff size={16} />}
      label={t`Hide from Browse`}
      onClick={run(() => {
        HideBrowseMod(game, { source: item.source, id: item.id, name: item.name })
          .then(() => useBrowseView.getState().noteHide())
          .catch(reportError(t`Could not hide that mod`))
      })}
    />,
    ...(item.author
      ? [
          <MenuAction
            key="hide-author"
            icon={<EyeOff size={16} />}
            label={t`Hide everything by ${{ author: item.author }}`}
            onClick={run(() => {
              HideBrowseAuthor(game, item.author)
                .then(() => useBrowseView.getState().noteHide())
                .catch(reportError(t`Could not hide that author`))
            })}
          />,
        ]
      : []),
  ]
}

type MenuState = NonNullable<ReturnType<typeof useBrowseSelection.getState>['menu']>

// Stays mounted after the menu closes so the known-broken confirmation an Add opens has somewhere to render.
function MenuHost({
  menu,
  open,
  card,
  sourceNames,
  close,
}: {
  menu: MenuState
  open: boolean
  card: ActionProps
  sourceNames: Map<string, string>
  close: () => void
}) {
  const cardAction = useCardAction(card, menu.item, menu.source)
  const { anchor } = menu
  const position =
    'el' in anchor ? {} : { anchorReference: 'anchorPosition' as const, anchorPosition: anchor }
  return (
    <>
      <Menu
        open={open}
        onClose={close}
        anchorEl={'el' in anchor ? anchor.el : undefined}
        {...position}
      >
        <Items
          item={menu.item}
          source={menu.source}
          card={card}
          sourceNames={sourceNames}
          close={close}
          cardAction={cardAction}
        />
      </Menu>
      {cardAction.dialog}
    </>
  )
}

/** The right-click menu of a Browse hit, in the Mods tab's menu style. */
export const BrowseMenu = memo(function BrowseMenuView({
  card,
  sourceNames,
}: {
  card: ActionProps
  sourceNames: Map<string, string>
}) {
  const menu = useBrowseSelection((s) => s.menu)
  const close = useBrowseSelection((s) => s.closeMenu)
  const [last, setLast] = useState(menu)
  if (menu && menu !== last) {
    setLast(menu)
  }
  const shown = menu ?? last
  if (!shown) {
    return null
  }
  return (
    <MenuHost
      key={`${shown.item.source}:${shown.item.id}:${shown.source}`}
      menu={shown}
      open={menu !== null}
      card={card}
      sourceNames={sourceNames}
      close={close}
    />
  )
})
