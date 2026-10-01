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
} from '@mui/material'
import { Pencil, Trash2 } from 'lucide-react'
import { useState } from 'react'
import type { Tool } from '../../bindings/github.com/Rethunk-AI/mortar/internal/tools/models.ts'
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
  const [editing, setEditing] = useState<Tool | null>(null)

  return (
    <>
      <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth={true} transitionDuration={0}>
        <DialogTitle>{t`Manage tools`}</DialogTitle>
        <DialogContent>
          <List dense={true}>
            {tools.map((tool) => (
              <ListItem
                key={tool.id}
                secondaryAction={
                  <>
                    <IconButton
                      edge="end"
                      aria-label={t`Edit ${tool.name}`}
                      onClick={() => setEditing(tool)}
                    >
                      <Pencil size={16} />
                    </IconButton>
                    <IconButton
                      edge="end"
                      aria-label={t`Remove ${tool.name}`}
                      onClick={() => remove(game, tool.id).catch(reportUnexpected)}
                    >
                      <Trash2 size={16} />
                    </IconButton>
                  </>
                }
              >
                <ListItemText primary={tool.name} secondary={tool.executable} />
              </ListItem>
            ))}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>{t`Close`}</Button>
        </DialogActions>
      </Dialog>
      <ToolEditorDialog
        open={editing !== null}
        initial={editing}
        onClose={() => setEditing(null)}
        onSave={(tool) => update(game, tool)}
      />
    </>
  )
}
