import { useLingui } from '@lingui/react/macro'
import {
  Account,
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/source/modio/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { SettingsSection } from '../SettingsSection.tsx'
import { Connection } from './SourceConnection.tsx'
import { TokenAccount } from './TokenAccount.tsx'

export function ModioAccount() {
  const { t } = useLingui()
  return (
    <SettingsSection>
      <TokenAccount
        source="modio"
        load={Account}
        signIn={SignIn}
        signOut={SignOut}
        fieldLabel={t`Access token`}
        description={t`Kept in your system keyring. Create a read-only personal access token at ${{ page: `${sourceLabel('modio')}/me/access` }}.`}
        removeLabel={t`Remove token`}
        keptStatus={t`Access token in your system keyring`}
        rejected={t`${{ source: sourceLabel('modio') }} rejected your access token`}
      >
        <Connection source="modio" />
      </TokenAccount>
    </SettingsSection>
  )
}
