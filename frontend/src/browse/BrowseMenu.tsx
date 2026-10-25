import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { ExternalLink, EyeOff, Info, Link2, PackagePlus, Plus } from 'lucide-react'
import { memo } from 'react'
import { AddDependency } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import {
  HideBrowseAuthor,
  HideBrowseMod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { copyText } from '../share/copyText.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { NEXUS, THUNDERSTORE } from './browseConstants.ts'
import type { BrowseItem } from './browseTypes.ts'
import { useBrowseSelection } from './selection.ts'
import { type ActionProps, useCardAction } from './useCardAction.tsx'
import { useBrowseView } from './view.ts'

function Items({
  item,
  source,
  card,
  sourceNames,
  close,
}: {
  item: BrowseItem
  source: string
  card: ActionProps
  sourceNames: Map<string, string>
  close: () => void
}) {
  const { t } = useLingui()
  const game = useCurrentGame()
  const { shownItem, url, installed, add } = useCardAction(card, item, source)
  const openId = useProfiles((s) => s.openId)
  const projects = (useSettings((s) => s.games?.[game]?.devProjects) ?? []).filter(
    (p) => p.profile === openId,
  )
  // A development profile's mod declares what it needs: the package joins the profile and the project's manifest.
  const depend = (dir: string, name: string) => {
    if (!installed) {
      add()
    }
    AddDependency(game, dir, item.id, item.version)
      .then(() =>
        useToasts.getState().push({
          kind: 'success',
          title: t`${{ mod: item.name }} is a dependency of ${{ name }}`,
        }),
      )
      .catch(reportError(t`Could not add the dependency`))
  }
  const choices = [{ source: item.source, url: item.url }, ...(item.alts ?? [])]
  const run = (fn: () => void) => () => {
    close()
    fn()
  }
  const opensFiles = source === NEXUS && !card.premium
  let addWhy: string | undefined
  if (installed) {
    addWhy = t`Already in this profile.`
  } else if (shownItem.bundled) {
    addWhy = t`Mortar installs this itself.`
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
    ...(item.source === THUNDERSTORE && item.version
      ? projects.map((p) => (
          <MenuAction
            key={`depend-${p.dir}`}
            icon={<PackagePlus size={16} />}
            label={t`Add as dependency of ${{ name: p.name }}`}
            onClick={run(() => depend(p.dir, p.name))}
          />
        ))
      : []),
    ...choices.map((c) => (
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
  if (!menu) {
    return null
  }
  const { anchor } = menu
  const position =
    'el' in anchor ? {} : { anchorReference: 'anchorPosition' as const, anchorPosition: anchor }
  return (
    <Menu
      open={true}
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
      />
    </Menu>
  )
})
