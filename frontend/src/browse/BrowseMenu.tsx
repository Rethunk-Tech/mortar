import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { ExternalLink, Info, Link2, Plus } from 'lucide-react'
import { copyText } from '../share/copyText.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { NEXUS } from './browseConstants.ts'
import type { BrowseItem } from './browseTypes.ts'
import { useBrowseSelection } from './selection.ts'
import { type ActionProps, useCardAction } from './useCardAction.tsx'

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
  const { shownItem, url, installed, add } = useCardAction(card, item, source)
  const choices = [{ source: item.source, url: item.url }, ...(item.alts ?? [])]
  const run = (fn: () => void) => () => {
    close()
    fn()
  }
  const opensFiles = source === NEXUS && !card.premium
  let addWhy: string | undefined
  if (installed) {
    addWhy = t`Already in this profile.`
  } else if (shownItem.bundled || shownItem.loader) {
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
        tooltip={addWhy}
        onClick={run(add)}
      />
    ),
    ...choices.map((c) => (
      <MenuAction
        key={`open-${c.source}`}
        icon={<ExternalLink size={16} />}
        label={t`Open page on ${sourceNames.get(c.source) ?? c.source}`}
        onClick={run(() => card.openUrl(c.url))}
      />
    )),
    <MenuAction
      key="copy"
      icon={<Link2 size={16} />}
      label={t`Copy link`}
      onClick={run(() => {
        void copyText(url, t`Link copied`)
      })}
    />,
  ]
}

/** The right-click menu of a Browse hit, in the Mods tab's menu style. */
export function BrowseMenu({
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
}
