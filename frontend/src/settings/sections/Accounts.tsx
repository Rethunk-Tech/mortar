import { useLingui } from '@lingui/react/macro'
import { Alert, Box } from '@mui/material'
import { useEffect, useState } from 'react'
import { KeysUnencrypted } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { refresh } from '../../shell/offline.ts'
import { githubLoggedIn } from '../../shell/sharedReads.ts'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { SettingsSection } from '../SettingsSection.tsx'
import { useSettingsRelevant } from '../useSettingsSearch.ts'
import { AccountRow, ServiceBlock } from './AccountRow.tsx'
import { ItchAccount } from './ItchAccount.tsx'
import { NexusMods } from './NexusMods.tsx'
import { Connection, ConnectionAccountRow, Quota } from './SourceConnection.tsx'

const NO_ACCOUNT = ['thunderstore', 'modrinth', 'curseforge', 'vsmoddb', 'paradox']

// The login answer lands on the block alone, not the page around it.
function GitHubAccount() {
  const { t } = useLingui()
  const [loggedIn, setLoggedIn] = useState<boolean | null>(null)
  const relevant = useSettingsRelevant(
    sourceLabel('github'),
    `${t`Your gh login`} · ${t`higher rate limit`}`,
    t`Requests`,
  )
  useEffect(() => {
    if (relevant) {
      githubLoggedIn().then(setLoggedIn).catch(reportUnexpected)
    }
  }, [relevant])
  let status = ''
  if (loggedIn) {
    status = `${t`Your gh login`} · ${t`higher rate limit`}`
  } else if (loggedIn === false) {
    status = t`No login, so the lower rate limit applies. Run gh auth login to raise it.`
  }
  return (
    <SettingsSection>
      <ServiceBlock>
        <AccountRow source="github" tone={loggedIn ? 'ok' : 'none'} status={status} />
        <Connection source="github" />
        <Quota source="github" />
      </ServiceBlock>
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
  const { t } = useLingui()
  const relevant = useSettingsRelevant(t`Connection`)
  useEffect(() => {
    if (relevant) {
      refresh().catch(reportUnexpected)
    }
  }, [relevant])
  // Services sit further apart than the rows inside one, so each block reads as its own; the sites with no account
  // are one row each and share a block.
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: `calc(${space.gutter} * 2)` }}>
      <UnencryptedKeys />
      <NexusMods />
      <GitHubAccount />
      <ItchAccount />
      <SettingsSection>
        {NO_ACCOUNT.map((id) => (
          <ConnectionAccountRow key={id} source={id} />
        ))}
      </SettingsSection>
    </Box>
  )
}
