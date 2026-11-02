import { useLingui } from '@lingui/react/macro'
import { Play, Settings2, Wrench } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Tool } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/tools/models.ts'
import { MenuAction, MenuNote } from '../shell/MenuAction.tsx'
import { MenuHeading, MenuRule } from '../shell/TitleMenu.tsx'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useTools } from '../tools/store.ts'
import { ToolsManageDialog } from '../tools/ToolsManageDialog.tsx'
import { HomeMenuButton } from './HomeButton.tsx'
import { ShortcutMenuItems } from './ProfileMenuItems.tsx'

const NO_TOOLS: Tool[] = []

export function LaunchMenu({ game, profile }: { game: string; profile: Profile }) {
  const { t } = useLingui()
  const [manageOpen, setManageOpen] = useState(false)
  const tools = useTools((s) => (s.loadedGame === game ? s.tools : NO_TOOLS))
  const load = useTools((s) => s.load)
  const launch = useTools((s) => s.launch)
  useEffect(() => {
    if (game) {
      load(game).catch(reportUnexpected)
    }
  }, [game, load])
  const items = (close: () => void) => [
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
  ]
  const manage = useMemo(
    () => <ToolsManageDialog game={game} open={manageOpen} onClose={() => setManageOpen(false)} />,
    [game, manageOpen],
  )
  return (
    <>
      <HomeMenuButton
        icon={<Wrench size={16} />}
        label={t`Tools`}
        menu={items}
        // Read again as the menu opens, so a tool added from the command line is listed.
        onOpen={() => {
          if (game) {
            load(game).catch(reportUnexpected)
          }
        }}
      />
      {manage}
    </>
  )
}
