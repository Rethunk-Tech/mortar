import { useLingui } from '@lingui/react/macro'
import { PrefKeys } from '../PrefRow.tsx'
import { SettingsSection } from '../SettingsSection.tsx'

export function Notifications() {
  const { t } = useLingui()
  return (
    <>
      <SettingsSection title={t`Downloads and runs`}>
        <PrefKeys keys={['notifyDownloadFinished', 'notifyDownloadFailed', 'notifyRunCrashed']} />
      </SettingsSection>
      <SettingsSection title={t`Mod updates`}>
        <PrefKeys keys={['updateDigest', 'notifyModUpdates']} />
      </SettingsSection>
    </>
  )
}
