import { useLingui } from '@lingui/react/macro'
import { ListItemText, Menu, MenuItem } from '@mui/material'
import { ChevronDown, Play, Rocket, Settings2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
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
  const close = () => setAnchor(null)
  return (
    <>
      <HomeButton
        icon={<Rocket size={16} />}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
      >
        {t`Launch`}
        <ChevronDown size={14} aria-hidden={true} />
      </HomeButton>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close} keepMounted={true}>
        <MenuHeading>{t`Run with this profile`}</MenuHeading>
        {tools.length === 0 ? (
          <MenuItem disabled={true} dense={true} sx={{ opacity: 1 }}>
            <ListItemText
              primary={t`No tools yet`}
              slotProps={{ primary: { sx: { fontSize: 14, color: 'text.primary' } } }}
            />
          </MenuItem>
        ) : null}
        {tools.map((tool) => (
          <MenuAction
            key={tool.id}
            icon={<Play size={16} />}
            label={tool.name}
            onClick={() => {
              close()
              const start = (): void => {
                launch(game, profile.id, tool.id).catch(
                  reportError(t`Could not start ${tool.name}`, start),
                )
              }
              start()
            }}
          />
        ))}
        <MenuRule />
        <MenuAction
          icon={<Settings2 size={16} />}
          label={t`Manage tools…`}
          onClick={() => {
            close()
            setManageOpen(true)
          }}
        />
        <MenuRule />
        <MenuHeading>{t`Shortcuts`}</MenuHeading>
        <ShortcutMenuItems profile={profile} close={close} />
      </Menu>
      <ToolsManageDialog game={game} open={manageOpen} onClose={() => setManageOpen(false)} />
    </>
  )
}
