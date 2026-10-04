import { useLingui } from '@lingui/react/macro'
import { Divider, ListItemText, Menu, MenuItem } from '@mui/material'
import { Play, Plus, Settings2, Wrench } from 'lucide-react'
import { useEffect, useState } from 'react'
import { IconAction } from '../shell/IconAction.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useTools } from './store.ts'
import { ToolEditorDialog } from './ToolEditorDialog.tsx'
import { ToolsManageDialog } from './ToolsManageDialog.tsx'

export function ToolsMenu({ game, profileID }: { game: string; profileID: string }) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [addOpen, setAddOpen] = useState(false)
  const [manageOpen, setManageOpen] = useState(false)
  const tools = useTools((s) => s.tools)
  const load = useTools((s) => s.load)
  const add = useTools((s) => s.add)
  const launch = useTools((s) => s.launch)

  useEffect(() => {
    if (game) {
      load(game).catch(reportUnexpected)
    }
  }, [game, load])

  const close = () => setAnchor(null)

  return (
    <>
      <IconAction
        label={t`Tools`}
        icon={<Wrench size={16} />}
        menu={true}
        onClick={(e) => setAnchor(e.currentTarget)}
      />
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close} transitionDuration={0}>
        {tools.length === 0 ? (
          <MenuItem disabled={true} dense={true} sx={{ opacity: 1 }}>
            <ListItemText
              primary={t`Launch other apps with this profile.`}
              slotProps={{
                primary: { sx: { fontSize: 12, color: 'text.secondary', fontWeight: 400 } },
              }}
            />
          </MenuItem>
        ) : null}
        {tools.length > 0 ? <Divider /> : null}
        {tools.map((tool) => (
          <MenuAction
            key={tool.id}
            icon={<Play size={16} />}
            label={tool.name}
            onClick={() => {
              close()
              launch(game, profileID, tool.id).catch(reportError(t`Could not start ${tool.name}`))
            }}
          />
        ))}
        {tools.length > 0 ? <Divider /> : null}
        <MenuAction
          icon={<Plus size={16} />}
          label={t`Add tool…`}
          onClick={() => {
            close()
            setAddOpen(true)
          }}
        />
        <MenuAction
          icon={<Settings2 size={16} />}
          label={t`Manage tools…`}
          onClick={() => {
            close()
            setManageOpen(true)
          }}
        />
      </Menu>
      <ToolEditorDialog
        open={addOpen}
        initial={null}
        onClose={() => setAddOpen(false)}
        onSave={async (tool) => {
          await add(game, tool)
        }}
      />
      <ToolsManageDialog game={game} open={manageOpen} onClose={() => setManageOpen(false)} />
    </>
  )
}
