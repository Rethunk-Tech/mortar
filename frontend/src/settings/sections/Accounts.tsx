import { useLingui } from '@lingui/react/macro'
import { Alert, Chip } from '@mui/material'
import { useEffect, useState } from 'react'
import { KeysUnencrypted } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { githubLoggedIn } from '../../shell/sharedReads.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettingsRelevant } from '../useSettingsSearch.ts'
import { ItchAccount } from './ItchAccount.tsx'
import { NexusMods } from './NexusMods.tsx'

// The login answer lands on the row alone, not the section around it.
function GitHubLogin() {
  const { t } = useLingui()
  const [loggedIn, setLoggedIn] = useState<boolean | null>(null)
  const label = t`${{ source: sourceLabel('github') }} CLI login`
  const using = t`Mortar uses your gh login for ${{ source: sourceLabel('github') }} requests, which raises the rate limit.`
  const without = t`Mortar sends ${{ source: sourceLabel('github') }} requests without a login. Run gh auth login to raise the rate limit.`
  const relevant = useSettingsRelevant(label, using, without)
  useEffect(() => {
    if (relevant) {
      githubLoggedIn().then(setLoggedIn).catch(reportUnexpected)
    }
  }, [relevant])
  return (
    <SettingRow label={label} description={loggedIn ? using : without}>
      {loggedIn === null ? null : (
        <Chip
          size="small"
          color={loggedIn ? 'primary' : 'default'}
          label={loggedIn ? t`In use` : t`Not signed in`}
        />
      )}
    </SettingRow>
  )
}

function GitHubAccount() {
  return (
    <SettingsSection title={sourceLabel('github')}>
      <GitHubLogin />
    </SettingsSection>
  )
}

function UnencryptedKeys() {
  const { t } = useLingui()
  const [plain, setPlain] = useState(false)
  useEffect(() => {
    KeysUnencrypted().then(setPlain).catch(reportUnexpected)
  }, [])
  if (!plain) {
    return null
  }
  return (
    <Alert variant="compact" severity="warning">
      {t`Mortar cannot reach a keyring, so your account keys are stored unencrypted on this computer.`}
    </Alert>
  )
}

export function Accounts() {
  return (
    <>
      <UnencryptedKeys />
      <NexusMods />
      <GitHubAccount />
      <ItchAccount />
    </>
  )
}
