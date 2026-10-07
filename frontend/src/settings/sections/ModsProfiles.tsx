import { useLingui } from '@lingui/react/macro'
import { SetEnableModsWhenInstalled } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { PrefSwitch } from '../PrefControls.tsx'
import { PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { SyncFolder } from './SyncFolder.tsx'

export function ModsProfiles() {
  const { t } = useLingui()
  const enableModsWhenInstalled = useSettings((s) => s.enableModsWhenInstalled)
  return (
    <SettingsSection title={t`Installing`}>
      <SettingRow label={t`Enable mods when installed`}>
        <PrefSwitch
          checked={enableModsWhenInstalled !== false}
          onChange={(on) =>
            persist(() => SetEnableModsWhenInstalled(on), t`Could not save that setting`)
          }
          label={t`Enable mods when installed`}
        />
      </SettingRow>
      <PrefKeys
        keys={[
          'reuseFomodChoices',
          'confirmRemovals',
          'driftChecks',
          'backgroundBadgeChecks',
          'showAdultContent',
        ]}
      />
      <SyncFolder />
    </SettingsSection>
  )
}
