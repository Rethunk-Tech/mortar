import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, TextField } from '@mui/material'
import { LogOut } from 'lucide-react'
import { type ReactNode, type SubmitEvent, useEffect, useId, useState } from 'react'
import { space } from '../../theme/density.ts'
import { errorKind } from '../../toasts/errorKind.ts'
import { type InlineError, inlineError, reportUnexpected } from '../../toasts/report.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { AccountRow, ServiceBlock } from './AccountRow.tsx'

interface Account {
  signedIn: boolean
  name: string
}

/** A source that signs in with one pasted token: its status row, and the masked field with Test until it is signed in.
 * The caller supplies the translated words, since each source names its token differently. */
export function TokenAccount({
  source,
  load,
  signIn,
  signOut,
  active = true,
  fieldLabel,
  description,
  removeLabel,
  keptStatus,
  rejected,
  children,
}: {
  source: string
  load: () => Promise<Account>
  signIn: (token: string) => Promise<Account>
  signOut: () => Promise<unknown>
  /** False leaves the account unread, for a card the settings search has hidden. */
  active?: boolean
  fieldLabel: string
  description: string
  removeLabel: string
  keptStatus: string
  rejected: string
  children?: ReactNode
}) {
  const { t } = useLingui()
  const tokenId = useId()
  const [name, setName] = useState<string | null>(null)
  const [token, setToken] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<InlineError | null>(null)
  useEffect(() => {
    if (active) {
      load()
        .then((a) => setName(a.signedIn ? a.name : ''))
        .catch(reportUnexpected)
    }
  }, [active, load])
  const test = (e: SubmitEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    signIn(token)
      .then((a) => {
        setToken('')
        setName(a.name)
      })
      .catch((err: unknown) =>
        setError(errorKind(err) === 'invalid' ? inlineError(err, rejected) : inlineError(err)),
      )
      .finally(() => setBusy(false))
  }
  let status = ''
  if (name) {
    status = `${name} · ${keptStatus}`
  } else if (name !== null) {
    status = t`Not signed in`
  }
  return (
    <ServiceBlock>
      <AccountRow
        source={source}
        tone={name ? 'ok' : 'none'}
        status={status}
        action={
          name ? (
            <Button
              variant="outlined"
              size="small"
              startIcon={<LogOut size={16} />}
              onClick={() =>
                signOut()
                  .then(() => setName(''))
                  .catch(reportUnexpected)
              }
            >
              {removeLabel}
            </Button>
          ) : null
        }
      />
      {name ? null : (
        <SettingRow label={fieldLabel} description={description}>
          <Box
            component="form"
            onSubmit={test}
            sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
              <TextField
                id={tokenId}
                slotProps={{ htmlInput: { 'aria-label': fieldLabel } }}
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
      {children}
    </ServiceBlock>
  )
}
