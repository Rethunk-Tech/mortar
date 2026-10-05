import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  TextField,
  Typography,
} from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { type SyntheticEvent, useEffect, useState } from 'react'
import { BugURL } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useBugReport } from './reportBug.ts'
import { saveDiagnostics } from './saveDiagnostics.ts'
import { useDiscardGuard } from './useDiscardGuard.tsx'

const empty = { title: '', happened: '', expected: '', steps: '' }

export function BugReportDialog() {
  const { t } = useLingui()
  const game = useBugReport((s) => s.game)
  const [form, setForm] = useState(empty)
  const [diagnostics, setDiagnostics] = useState(true)
  useEffect(() => {
    if (game !== null) {
      setForm(empty)
      setDiagnostics(true)
    }
  }, [game])
  const close = () => useBugReport.setState({ game: null })
  const guard = useDiscardGuard(
    Object.values(form).some((v) => v.trim() !== ''),
    close,
  )
  const field = (key: keyof typeof empty) => ({
    value: form[key],
    onChange: (e: { target: { value: string } }) => setForm({ ...form, [key]: e.target.value }),
  })
  const submit = (e: SyntheticEvent) => {
    e.preventDefault()
    if (form.happened.trim() === '') {
      return
    }
    // The zip is saved first so the GitHub page opens after the user has a file to drag into it.
    const attach = diagnostics ? saveDiagnostics(game ?? '', '', true) : Promise.resolve(false)
    attach
      .then(() => BugURL(game ?? '', { ...form, diagnostics }))
      .then((url) => Browser.OpenURL(url))
      .then(close, reportUnexpected)
  }
  return (
    <Dialog open={game !== null} onClose={guard.request} fullWidth={true} maxWidth="sm">
      <form onSubmit={submit}>
        <DialogTitle>{t`Report a bug`}</DialogTitle>
        <DialogContent
          sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '8px !important' }}
        >
          <TextField
            label={t`Title`}
            placeholder={t`A short summary`}
            fullWidth={true}
            {...field('title')}
          />
          <TextField
            label={t`What happened`}
            required={true}
            autoFocus={true}
            multiline={true}
            minRows={3}
            fullWidth={true}
            {...field('happened')}
          />
          <TextField
            label={t`What you expected`}
            multiline={true}
            minRows={2}
            fullWidth={true}
            {...field('expected')}
          />
          <TextField
            label={t`Steps to reproduce`}
            multiline={true}
            minRows={2}
            fullWidth={true}
            {...field('steps')}
          />
          <FormControlLabel
            control={
              <Checkbox checked={diagnostics} onChange={(e) => setDiagnostics(e.target.checked)} />
            }
            label={t`Attach diagnostics (a zip of checks, logs and settings with secrets and your home folder removed; you drag it into the issue)`}
          />
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
            {t`Opens a GitHub issue with these filled in. You review it before posting.`}
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={guard.request}>{t`Cancel`}</Button>
          <Button type="submit" variant="contained" disabled={form.happened.trim() === ''}>
            {t`Open on GitHub`}
          </Button>
        </DialogActions>
      </form>
      {guard.dialog}
    </Dialog>
  )
}
