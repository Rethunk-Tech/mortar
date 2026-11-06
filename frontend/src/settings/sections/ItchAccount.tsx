import { useLingui } from '@lingui/react/macro'
import {
  SignIn,
  SignOut,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/source/itch/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { itchAccount } from '../../shell/sharedReads.ts'
import { SettingsSection } from '../SettingsSection.tsx'
import { useSettingsRelevant } from '../useSettingsSearch.ts'
import { Connection } from './SourceConnection.tsx'
import { TokenAccount } from './TokenAccount.tsx'

export function ItchAccount() {
  const { t } = useLingui()
  const description = t`Kept in your system keyring. Create one on ${{ source: sourceLabel('itch') }} under Settings, API keys.`
  const relevant = useSettingsRelevant(sourceLabel('itch'), t`Remove key`, t`API key`, description)
  return (
    <SettingsSection>
      <TokenAccount
        source="itch"
        active={relevant}
        load={itchAccount}
        signIn={SignIn}
        signOut={SignOut}
        fieldLabel={t`API key`}
        description={description}
        removeLabel={t`Remove key`}
        keptStatus={t`API key in your system keyring`}
        rejected={t`${{ source: sourceLabel('itch') }} rejected your API key`}
      >
        <Connection source="itch" />
      </TokenAccount>
    </SettingsSection>
  )
}
