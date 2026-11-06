import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, TextField } from '@mui/material'
import { LogOut } from 'lucide-react'
import { type SubmitEvent, useEffect, useId, useState } from 'react'
import {
  Account,
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/source/modio/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { space } from '../../theme/density.ts'
import { errorKind } from '../../toasts/errorKind.ts'
import { type InlineError, inlineError, reportUnexpected } from '../../toasts/report.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { AccountRow, ServiceBlock } from './AccountRow.tsx'

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
            ? inlineError(err, t`${{ source: sourceLabel('modio') }} rejected your access token`)
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
  let status = ''
  if (name) {
    status = `${name} · ${t`Access token in your system keyring`}`
  } else if (name !== null) {
    status = t`Not signed in`
  }
  return (
    <ServiceBlock>
      <AccountRow
        source="modio"
        tone={name ? 'ok' : 'none'}
        status={status}
        action={
          name ? (
            <Button
              variant="outlined"
              size="small"
              startIcon={<LogOut size={16} />}
              onClick={signOut}
            >
              {t`Remove token`}
            </Button>
          ) : null
        }
      />
      {name ? null : (
        <SettingRow
          label={t`Access token`}
          description={t`Kept in your system keyring. Create a read-only personal access token at ${{ page: `${sourceLabel('modio')}/me/access` }}.`}
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
      )}
    </ServiceBlock>
  )
}

export function ModioAccount() {
  return (
    <SettingsSection>
      <ModioRow />
    </SettingsSection>
  )
}
