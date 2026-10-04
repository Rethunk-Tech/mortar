import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField } from '@mui/material'
import { useEffect, useState } from 'react'
import { PickExecutable } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type { Tool } from '../../bindings/github.com/Rethunk-AI/mortar/internal/tools/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'

// The tokens Mortar substitutes in a tool's arguments; passed as a value so Lingui does not read them as variables.
const placeholders = '{game}, {mods}, {saves}, {profile}'

function argsToText(args: string[] | null | undefined): string {
  return (args ?? []).join('\n')
}

function textToArgs(text: string): string[] {
  const lines = text.split('\n')
  const out: string[] = []
  for (const line of lines) {
    if (line.length > 0) {
      out.push(line)
    }
  }
  return out
}

interface Props {
  open: boolean
  initial: Tool | null
  onClose: () => void
  onSave: (tool: Tool) => Promise<void>
}

export function ToolEditorDialog({ open, initial, onClose, onSave }: Props) {
  const { t } = useLingui()
  const [name, setName] = useState('')
  const [executable, setExecutable] = useState('')
  const [argsText, setArgsText] = useState('')
  const [workingDir, setWorkingDir] = useState('')
  const [saving, run] = usePending()

  useEffect(() => {
    if (!open) {
      return
    }
    setName(initial?.name ?? '')
    setExecutable(initial?.executable ?? '')
    setArgsText(argsToText(initial?.arguments))
    setWorkingDir(initial?.workingDir ?? '')
  }, [open, initial])

  const browse = () => {
    PickExecutable(t`Choose executable`)
      .then((path) => {
        if (path) {
          setExecutable(path)
        }
      })
      .catch(reportUnexpected)
  }

  const save = () => {
    const tool: Tool = {
      id: initial?.id ?? '',
      name: name.trim(),
      executable: executable.trim(),
      arguments: textToArgs(argsText),
      workingDir: workingDir.trim(),
    }
    run(() => onSave(tool).then(() => onClose()), { errorTitle: t`Could not save the tool` })
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth={true} transitionDuration={0}>
      <form
        onSubmit={(e) => {
          e.preventDefault()
          if (saving || !name.trim() || !executable.trim()) {
            return
          }
          save()
        }}
      >
        <DialogTitle>{initial ? t`Edit tool` : t`Add tool`}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
          <TextField
            label={t`Name`}
            value={name}
            onChange={(e) => setName(e.target.value)}
            fullWidth={true}
          />
          <TextField
            label={t`Executable`}
            value={executable}
            onChange={(e) => setExecutable(e.target.value)}
            fullWidth={true}
          />
          <Button
            type="button"
            variant="outlined"
            onClick={browse}
            sx={{ alignSelf: 'flex-start' }}
          >
            {t`Browse…`}
          </Button>
          <TextField
            label={t`Arguments`}
            value={argsText}
            onChange={(e) => setArgsText(e.target.value)}
            fullWidth={true}
            multiline={true}
            minRows={2}
            helperText={t`One argument per line. Placeholders: ${placeholders}.`}
          />
          <TextField
            label={t`Working directory`}
            value={workingDir}
            onChange={(e) => setWorkingDir(e.target.value)}
            fullWidth={true}
            helperText={t`Leave empty to use the game folder. Same placeholders as arguments.`}
          />
        </DialogContent>
        <DialogActions>
          <Button type="button" onClick={onClose}>
            {t`Cancel`}
          </Button>
          <Button
            type="submit"
            variant="contained"
            disabled={saving || !name.trim() || !executable.trim()}
          >
            {t`Save`}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}
