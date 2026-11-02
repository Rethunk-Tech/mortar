import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Chip, TextField } from '@mui/material'
import { type SubmitEvent, useEffect, useId, useState } from 'react'
import {
  Account,
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/source/modio/service.ts'
import { space } from '../../theme/density.ts'
import { errorKind } from '../../toasts/errorKind.ts'
import { type InlineError, inlineError, reportUnexpected } from '../../toasts/report.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'

// The account answer lands on the row alone, not the section around it.
function ModioRow() {
  const { t } = useLingui()
  const tokenId = useId()
  const [name, setName] = useState<string | null>(null)
  const [token, setToken] = useState('')
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
    SignIn(token)
      .then((a) => {
        setToken('')
        setName(a.name)
      })
      .catch((err: unknown) =>
        setError(
          errorKind(err) === 'invalid'
            ? inlineError(err, t`mod.io rejected your access token`)
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
  return name ? (
    <SettingRow
      label={t`Account`}
      description={t`Your access token is kept in your system keyring.`}
    >
      <Chip size="small" color="primary" label={name} />
      <Button size="small" onClick={signOut}>
        {t`Remove token`}
      </Button>
    </SettingRow>
  ) : (
    <SettingRow
      label={t`Access token`}
      description={t`Kept in your system keyring. Create a read-only OAuth token at mod.io/me/access.`}
    >
      <Box
        component="form"
        onSubmit={test}
        sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
          <TextField
            id={tokenId}
            slotProps={{ htmlInput: { 'aria-label': t`Access token` } }}
            type="password"
            size="small"
            autoComplete="off"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            sx={{ width: 260, maxWidth: '100%' }}
          />
          <Button
            type="submit"
            variant="outlined"
            disabled={busy || token.trim() === ''}
            sx={{ flexShrink: 0 }}
          >
            {t`Test`}
          </Button>
        </Box>
        {error ? (
          <Alert variant="compact" severity="error" title={error.details}>
            {error.message}
          </Alert>
        ) : null}
      </Box>
    </SettingRow>
  )
}

export function ModioAccount() {
  const { t } = useLingui()
  return (
    <SettingsSection title={t`mod.io`}>
      <ModioRow />
    </SettingsSection>
  )
}
