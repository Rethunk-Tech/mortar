import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  List,
  ListItem,
  ListItemText,
  Tooltip,
} from '@mui/material'
import { Inbox, Pencil, Trash2 } from 'lucide-react'
import { useState } from 'react'
import type { Tool } from '../../bindings/github.com/Rethunk-AI/mortar/internal/tools/models.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useTools } from './store.ts'
import { ToolEditorDialog } from './ToolEditorDialog.tsx'

export function ToolsManageDialog({
  game,
  open,
  onClose,
}: {
  game: string
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const tools = useTools((s) => s.tools)
  const remove = useTools((s) => s.remove)
  const update = useTools((s) => s.update)
  const add = useTools((s) => s.add)
  const [editing, setEditing] = useState<Tool | null>(null)
  const [adding, setAdding] = useState(false)

  return (
    <>
      <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth={true} transitionDuration={0}>
        <DialogTitle>{t`Manage tools`}</DialogTitle>
        <DialogContent>
          {tools.length === 0 ? (
            <EmptyState compact={true} icon={<Inbox size={28} />} title={t`No tools yet`}>
              {t`Add a program you run alongside the game, such as a save editor or a map viewer.`}
            </EmptyState>
          ) : null}
          <List dense={true}>
            {tools.map((tool) => (
              <ListItem
                key={tool.id}
                secondaryAction={
                  <>
                    <Tooltip title={t`Edit ${tool.name}`}>
                      <IconButton
                        edge="end"
                        aria-label={t`Edit ${tool.name}`}
                        onClick={() => setEditing(tool)}
                      >
                        <Pencil size={16} />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title={t`Remove ${tool.name}`}>
                      <IconButton
                        edge="end"
                        aria-label={t`Remove ${tool.name}`}
                        onClick={() => remove(game, tool.id).catch(reportUnexpected)}
                      >
                        <Trash2 size={16} />
                      </IconButton>
                    </Tooltip>
                  </>
                }
              >
                <ListItemText primary={tool.name} secondary={tool.executable} />
              </ListItem>
            ))}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAdding(true)}>{t`Add tool…`}</Button>
          <Button onClick={onClose}>{t`Close`}</Button>
        </DialogActions>
      </Dialog>
      <ToolEditorDialog
        open={adding}
        initial={null}
        onClose={() => setAdding(false)}
        onSave={async (tool) => {
          await add(game, tool)
        }}
      />
      <ToolEditorDialog
        open={editing !== null}
        initial={editing}
        onClose={() => setEditing(null)}
        onSave={(tool) => update(game, tool)}
      />
    </>
  )
}
