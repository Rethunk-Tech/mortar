import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Chip, TextField } from '@mui/material'
import { type SubmitEvent, useEffect, useId, useState } from 'react'
import {
  Account,
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/source/itch/service.ts'
import { errorKind } from '../../toasts/errorKind.ts'
import { type InlineError, inlineError, reportUnexpected } from '../../toasts/report.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'

export function ItchAccount() {
  const { t } = useLingui()
  const keyId = useId()
  const [name, setName] = useState<string | null>(null)
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<InlineError | null>(null)
  useEffect(() => {
    Account()
      .then((a) => setName(a.signedIn ? a.name : ''))
      .catch(reportUnexpected)
  }, [])
  const test = (e: SubmitEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    SignIn(key)
      .then((a) => {
        setKey('')
        setName(a.name)
      })
      .catch((err: unknown) =>
        setError(
          errorKind(err) === 'invalid'
            ? inlineError(err, t`itch.io rejected your API key`)
            : inlineError(err),
        ),
      )
      .finally(() => setBusy(false))
  }
  const signOut = () => {
    SignOut()
      .then(() => setName(''))
      .catch(reportUnexpected)
  }
  return (
    <SettingsSection title={t`itch.io`}>
      {name ? (
        <SettingRow
          label={t`Account`}
          description={t`Your API key is kept in your system keyring.`}
        >
          <Chip size="small" color="primary" label={name} />
          <Button size="small" onClick={signOut}>
            {t`Remove key`}
          </Button>
        </SettingRow>
      ) : (
        <Box
          component="form"
          onSubmit={test}
          sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}
        >
          <Box component="label" htmlFor={keyId} sx={{ fontSize: 14, fontWeight: 600 }}>
            {t`API key`}
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
            <TextField
              id={keyId}
              type="password"
              size="small"
              autoComplete="off"
              value={key}
              onChange={(e) => setKey(e.target.value)}
              helperText={t`Kept in your system keyring. Create one on itch.io under Settings, API keys.`}
              sx={{ width: 420, maxWidth: '100%' }}
            />
            <Button
              type="submit"
              variant="contained"
              disabled={busy || key.trim() === ''}
              sx={{ flexShrink: 0, height: 40 }}
            >
              {t`Test`}
            </Button>
          </Box>
          {error ? (
            <Alert severity="error" title={error.details}>
              {error.message}
            </Alert>
          ) : null}
        </Box>
      )}
    </SettingsSection>
  )
}
