import { useLingui } from '@lingui/react/macro'
import { Divider, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { Play, Plus, Settings2, Wrench } from 'lucide-react'
import { useEffect, useState } from 'react'
import { IconAction } from '../shell/IconAction.tsx'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
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
  const push = useToasts((s) => s.push)

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
        {tools.map((tool) => (
          <MenuItem
            key={tool.id}
            onClick={() => {
              close()
              launch(game, profileID, tool.id)
                .then(() => push({ kind: 'success', title: t`Started ${tool.name}` }))
                .catch((e: unknown) =>
                  push({
                    kind: 'error',
                    title: t`Could not start ${tool.name}`,
                    body: errorMessage(e),
                  }),
                )
            }}
          >
            <ListItemIcon sx={{ color: 'inherit' }}>
              <Play size={16} />
            </ListItemIcon>
            <ListItemText>{tool.name}</ListItemText>
          </MenuItem>
        ))}
        {tools.length > 0 ? <Divider /> : null}
        <MenuItem
          onClick={() => {
            close()
            setAddOpen(true)
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <Plus size={16} />
          </ListItemIcon>
          <ListItemText>{t`Add tool…`}</ListItemText>
        </MenuItem>
        <MenuItem
          onClick={() => {
            close()
            setManageOpen(true)
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <Settings2 size={16} />
          </ListItemIcon>
          <ListItemText>{t`Manage tools`}</ListItemText>
        </MenuItem>
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
