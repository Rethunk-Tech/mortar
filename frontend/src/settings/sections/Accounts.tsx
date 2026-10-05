import { useLingui } from '@lingui/react/macro'
import { Chip } from '@mui/material'
import { useEffect, useState } from 'react'
import { GitHubLoggedIn } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { ItchAccount } from './ItchAccount.tsx'
import { NexusMods } from './NexusMods.tsx'

function GitHubAccount() {
  const { t } = useLingui()
  const [loggedIn, setLoggedIn] = useState<boolean | null>(null)
  useEffect(() => {
    GitHubLoggedIn().then(setLoggedIn).catch(reportUnexpected)
  }, [])
  return (
    <SettingsSection title={t`GitHub`}>
      <SettingRow
        label={t`GitHub CLI login`}
        description={
          loggedIn
            ? t`Mortar uses your gh login for GitHub requests, which raises the rate limit.`
            : t`Mortar sends GitHub requests without a login. Run gh auth login to raise the rate limit.`
        }
      >
        {loggedIn === null ? null : (
          <Chip
            size="small"
            color={loggedIn ? 'primary' : 'default'}
            label={loggedIn ? t`In use` : t`Not signed in`}
          />
        )}
      </SettingRow>
    </SettingsSection>
  )
}

export function Accounts() {
  return (
    <>
      <NexusMods />
      <GitHubAccount />
      <ItchAccount />
    </>
  )
}
