import { useLingui } from '@lingui/react/macro'
import { Switch } from '@mui/material'
import { SetEnableModsWhenInstalled } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

export function ModsProfiles() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const enableModsWhenInstalled = useSettings((s) => s.enableModsWhenInstalled)
  return (
    <SettingsSection title={t`Installing`}>
      <SettingRow label={t`Enable mods when installed`}>
        <Switch
          checked={enableModsWhenInstalled !== false}
          onChange={(_, on) =>
            persist(() => SetEnableModsWhenInstalled(on), push, t`Couldn't save that setting`)
          }
        />
      </SettingRow>
      <PrefKeys
        keys={['reuseFomodChoices', 'confirmRemovals', 'driftChecks', 'backgroundBadgeChecks']}
      />
    </SettingsSection>
  )
}
