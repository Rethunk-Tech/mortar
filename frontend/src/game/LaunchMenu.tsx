import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { ChevronDown, Play, Settings2, Wrench } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { MenuAction, MenuNote } from '../shell/MenuAction.tsx'
import { MenuHeading, MenuRule } from '../shell/TitleMenu.tsx'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useTools } from '../tools/store.ts'
import { ToolsManageDialog } from '../tools/ToolsManageDialog.tsx'
import { HomeButton } from './HomeButton.tsx'
import { ShortcutMenuItems } from './ProfileMenuItems.tsx'

export function LaunchMenu({ game, profile }: { game: string; profile: Profile }) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [manageOpen, setManageOpen] = useState(false)
  const tools = useTools((s) => s.tools)
  const load = useTools((s) => s.load)
  const launch = useTools((s) => s.launch)
  useEffect(() => {
    if (game) {
      load(game).catch(reportUnexpected)
    }
  }, [game, load])
  const close = useCallback(() => setAnchor(null), [])
  // Built once per tool list and profile: opening and closing the menu re-renders this view, and the same items and
  // closed dialog let React skip them.
  const items = useMemo(
    () => [
      <MenuHeading key="run">{t`Run with this profile`}</MenuHeading>,
      tools.length === 0 ? <MenuNote key="none">{t`No tools yet`}</MenuNote> : null,
      ...tools.map((tool) => (
        <MenuAction
          key={tool.id}
          icon={<Play size={16} />}
          label={tool.name}
          onClick={() => {
            close()
            const start = (): void => {
              launch(game, profile.id, tool.id).catch(
                reportError(t`Could not start ${{ name: tool.name }}`, start),
              )
            }
            start()
          }}
        />
      )),
      <MenuRule key="rule-manage" />,
      <MenuAction
        key="manage"
        icon={<Settings2 size={16} />}
        label={t`Manage tools…`}
        onClick={() => {
          close()
          setManageOpen(true)
        }}
      />,
      <MenuRule key="rule-shortcuts" />,
      <MenuHeading key="shortcuts">{t`Shortcuts`}</MenuHeading>,
      <ShortcutMenuItems key="shortcut-items" profile={profile} close={close} />,
    ],
    [tools, game, profile, launch, close, t],
  )
  const manage = useMemo(
    () => <ToolsManageDialog game={game} open={manageOpen} onClose={() => setManageOpen(false)} />,
    [game, manageOpen],
  )
  return (
    <>
      <HomeButton
        icon={<Wrench size={16} />}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
      >
        {t`Tools`}
        <ChevronDown size={14} aria-hidden={true} />
      </HomeButton>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close}>
        {items}
      </Menu>
      {manage}
    </>
  )
}
