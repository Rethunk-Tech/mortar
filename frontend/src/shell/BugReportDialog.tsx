import { useLingui } from '@lingui/react/macro'
import {
  Checkbox,
  FormControlLabel,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import {
  BugText,
  BugURL,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { openPage } from '../mods/menu.ts'
import { openPageLabel } from '../mods/modActions.ts'
import { copyText } from '../share/copyText.ts'
import { usePending } from '../toasts/usePending.ts'
import { type Destination, GO, getDestination, REPORT_KEY, setDestination } from './destination.ts'
import { Field, FormDialog, TextInput } from './dialog/FormDialog.tsx'
import { openWeb } from './openWeb.ts'
import { useBugReport } from './reportBug.ts'
import { saveDiagnostics } from './saveDiagnostics.ts'

const empty = { title: '', happened: '', expected: '', steps: '' }

export function BugReportDialog() {
  const { t, i18n } = useLingui()
  const game = useBugReport((s) => s.game)
  const [form, setForm] = useState(empty)
  const [diagnostics, setDiagnostics] = useState(true)
  const [to, setTo] = useState<Destination>(() => getDestination(REPORT_KEY))
  const discord = to === 'discord'
  useEffect(() => {
    if (game !== null) {
      setForm(empty)
      setDiagnostics(true)
    }
  }, [game])
  const close = () => useBugReport.setState({ game: null })
  const field = (key: keyof typeof empty) => ({
    value: form[key],
    onChange: (value: string) => setForm({ ...form, [key]: value }),
  })
  const [busy, run] = usePending()
  const submit = () =>
    run(async () => {
      // The zip is saved first so the destination opens after the user has a file to attach to it.
      if (diagnostics) {
        await saveDiagnostics(game ?? '', '', true)
      }
      const report = { ...form, diagnostics }
      if (discord) {
        const text = await BugText(game ?? '', report)
        copyText(text, t`Report copied`, () => {
          openPage(`${GO}bug-reports/`)
          close()
        })
        return
      }
      await openWeb(await BugURL(game ?? '', report))
      close()
    })
  return (
    <FormDialog
      open={game !== null}
      onClose={close}
      title={t`Report a bug`}
      submitLabel={
        discord
          ? t`Copy and open ${{ app: sourceLabel('discord') }}`
          : openPageLabel(i18n, 'github')
      }
      issue={form.happened.trim() === '' ? t`Say what happened.` : ''}
      dirty={Object.values(form).some((v) => v.trim() !== '')}
      busy={busy}
      onSubmit={submit}
    >
      <Field label={t`Post on`}>
        <ToggleButtonGroup
          exclusive={true}
          size="small"
          fullWidth={true}
          value={to}
          aria-label={t`Post on`}
          onChange={(_, next: Destination | null) => {
            if (next !== null) {
              setTo(next)
              setDestination(REPORT_KEY, next)
            }
          }}
        >
          <ToggleButton value="discord">{sourceLabel('discord')}</ToggleButton>
          <ToggleButton value="github">{sourceLabel('github')}</ToggleButton>
        </ToggleButtonGroup>
      </Field>
      <TextInput label={t`Title`} placeholder={t`A short summary`} {...field('title')} />
      <TextInput
        label={t`What happened`}
        autoFocus={true}
        multiline={true}
        {...field('happened')}
      />
      <TextInput label={t`What you expected`} multiline={true} {...field('expected')} />
      <TextInput label={t`Steps to reproduce`} multiline={true} {...field('steps')} />
      <FormControlLabel
        control={
          <Checkbox checked={diagnostics} onChange={(e) => setDiagnostics(e.target.checked)} />
        }
        label={
          discord
            ? t`Attach diagnostics (a zip of checks, logs and settings with secrets and your home folder removed; you attach it to the post)`
            : t`Attach diagnostics (a zip of checks, logs and settings with secrets and your home folder removed; you drag it into the issue)`
        }
      />
      <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
        {discord
          ? t`Copies the report for you to paste into a new post in the bug-reports forum on Mortar's ${{ app: sourceLabel('discord') }}, and saves the diagnostics zip to attach.`
          : t`Opens a ${{ source: sourceLabel('github') }} issue with these filled in. You review it before posting.`}
      </Typography>
    </FormDialog>
  )
}
