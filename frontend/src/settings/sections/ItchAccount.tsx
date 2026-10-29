import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, TextField } from '@mui/material'
import { type SubmitEvent, useEffect, useId, useState } from 'react'
import {
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/source/itch/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { itchAccount } from '../../shell/sharedReads.ts'
import { space } from '../../theme/density.ts'
import { errorKind } from '../../toasts/errorKind.ts'
import { type InlineError, inlineError, reportUnexpected } from '../../toasts/report.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettingsRelevant } from '../useSettingsSearch.ts'
import { AccountRow, ServiceBlock } from './AccountRow.tsx'
import { Connection } from './SourceConnection.tsx'

export function ItchAccount() {
  const { t } = useLingui()
  const keyId = useId()
  const [name, setName] = useState<string | null>(null)
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<InlineError | null>(null)
  const relevant = useSettingsRelevant(
    sourceLabel('itch'),
    t`Remove key`,
    t`API key`,
    t`Kept in your system keyring. Create one on ${{ source: sourceLabel('itch') }} under Settings, API keys.`,
  )
  useEffect(() => {
    if (relevant) {
      itchAccount()
        .then((a) => setName(a.signedIn ? a.name : ''))
        .catch(reportUnexpected)
    }
  }, [relevant])
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
            ? inlineError(err, t`${{ source: sourceLabel('itch') }} rejected your API key`)
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
  let itchStatus = ''
  if (name) {
    itchStatus = `${name} · ${t`API key in your system keyring`}`
  } else if (name !== null) {
    itchStatus = t`Not signed in`
  }
  return (
    <SettingsSection>
      <ServiceBlock>
        <AccountRow
          source="itch"
          tone={name ? 'ok' : 'none'}
          status={itchStatus}
          action={
            name ? (
              <Button size="small" onClick={signOut}>
                {t`Remove key`}
              </Button>
            ) : null
          }
        />
        {name ? null : (
          <SettingRow
            label={t`API key`}
            description={t`Kept in your system keyring. Create one on ${{ source: sourceLabel('itch') }} under Settings, API keys.`}
          >
            <Box
              component="form"
              onSubmit={test}
              sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
                <TextField
                  id={keyId}
                  slotProps={{ htmlInput: { 'aria-label': t`API key` } }}
                  type="password"
                  size="small"
                  autoComplete="off"
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  sx={{ width: 260, maxWidth: '100%' }}
                />
                <Button
                  type="submit"
                  variant="outlined"
                  disabled={busy || key.trim() === ''}
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
        <Connection source="itch" />
      </ServiceBlock>
    </SettingsSection>
  )
}
