import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { Wrench } from 'lucide-react'
import { useEffect, useState } from 'react'
import { PickExecutable } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { Tool } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/tools/models.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { FormDialog, TextInput } from '../shell/dialog/FormDialog.tsx'
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

  const dirty =
    name !== (initial?.name ?? '') ||
    executable !== (initial?.executable ?? '') ||
    argsText !== argsToText(initial?.arguments) ||
    workingDir !== (initial?.workingDir ?? '')

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

  let issue = ''
  if (!name.trim()) {
    issue = t`Name the tool.`
  } else if (!executable.trim()) {
    issue = t`Choose executable`
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={initial ? t`Edit tool` : t`Add tool`}
      icon={
        <ToneIcon tone="neutral">
          <Wrench size={22} />
        </ToneIcon>
      }
      submitLabel={t`Save`}
      issue={issue}
      dirty={dirty}
      busy={saving}
      onSubmit={save}
    >
      <TextInput label={t`Name`} value={name} onChange={setName} />
      <TextInput label={t`Executable`} value={executable} onChange={setExecutable} />
      <Button type="button" variant="outlined" onClick={browse} sx={{ alignSelf: 'flex-start' }}>
        {t`Choose folder…`}
      </Button>
      <TextInput
        label={t`Arguments`}
        value={argsText}
        onChange={setArgsText}
        multiline={true}
        hint={t`One argument per line. Placeholders: ${placeholders}.`}
      />
      <TextInput
        label={t`Working directory`}
        value={workingDir}
        onChange={setWorkingDir}
        hint={t`Leave empty to use the game folder. Same placeholders as arguments.`}
      />
    </FormDialog>
  )
}
