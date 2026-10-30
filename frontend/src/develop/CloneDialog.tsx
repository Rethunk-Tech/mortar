import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField } from '@mui/material'
import { readStored, writeStored } from '@rethunk/react-load-hooks'
import { GitBranch } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Clone } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useProfiles } from '../profiles/store.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { Field, FormDialog, TextInput } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { isRemembered, REMEMBERED } from './draft.ts'
import { isRepoAddress } from './repo.ts'

/** Clones a GitHub mod project into the projects folder the New mod wizard remembers and adds it. */
export function CloneDialog({
  open,
  game,
  onClose,
}: {
  open: boolean
  game: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const [repo, setRepo] = useState('')
  const [parent, setParent] = useState('')
  const [pending, run] = usePending()
  useEffect(() => {
    if (open) {
      setRepo('')
      setParent(readStored(REMEMBERED, {}, isRemembered).parent ?? '')
    }
  }, [open])
  const pick = () => {
    PickFolder(t`Folder for your mod projects`)
      .then((dir) => {
        if (dir) {
          setParent(dir)
        }
      })
      .catch(reportUnexpected)
  }
  const clone = () =>
    run(
      async () => {
        const project = await Clone(game, repo.trim(), parent, openId)
        writeStored(REMEMBERED, { ...readStored(REMEMBERED, {}, isRemembered), parent })
        useToasts
          .getState()
          .push({ kind: 'success', title: t`Added ${{ name: project.name }}`, body: project.dir })
        onClose()
      },
      { errorTitle: t`Could not clone the project` },
    )
  let issue = ''
  if (!isRepoAddress(repo)) {
    issue = t`Name the repository as owner/name or its github.com address.`
  } else if (!parent) {
    issue = t`Choose the folder the project goes in.`
  }
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={t`Clone from ${{ source: sourceLabel('github') }}`}
      icon={
        <ToneIcon tone="neutral">
          <GitBranch size={22} />
        </ToneIcon>
      }
      submitLabel={t`Clone`}
      issue={issue}
      dirty={repo.trim() !== ''}
      busy={pending}
      onSubmit={clone}
    >
      <TextInput
        label={t`Repository`}
        value={repo}
        onChange={setRepo}
        placeholder="owner/name"
        autoFocus={true}
      />
      <Field label={t`Projects folder`}>
        <Box sx={{ display: 'flex', gap: space.gap }}>
          <TextField
            size="small"
            fullWidth={true}
            value={parent}
            slotProps={{ input: { readOnly: true } }}
          />
          <Button variant="outlined" onClick={pick}>
            {t`Choose…`}
          </Button>
        </Box>
      </Field>
    </FormDialog>
  )
}
