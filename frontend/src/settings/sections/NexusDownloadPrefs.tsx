import { useLingui } from '@lingui/react/macro'
import {
  SetAutoTrackNexus,
  SetNxmDefaultProfile,
  SetParallelDownloads,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useToasts } from '../../toasts/store.ts'
import { gamePrefs } from '../gamePrefs.ts'
import { PrefNumber, PrefSelect, PrefSwitch } from '../PrefControls.tsx'
import { persist } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

const defaultParallelDownloads = 3

export function NexusDownloadPrefs() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  const profiles = useProfiles((s) => s.profiles)
  return (
    <>
      <SettingRow
        label={t`Auto-track installed mods`}
        description={t`Track a Nexus mod when Mortar installs it`}
      >
        <PrefSwitch
          checked={useSettings((s) => s.autoTrackNexus)}
          onChange={(on) => persist(() => SetAutoTrackNexus(on), push, fail)}
        />
      </SettingRow>
      <SettingRow
        label={t`Parallel downloads`}
        description={t`Premium and GitHub downloads at once`}
      >
        <PrefNumber
          value={useSettings((s) => s.parallelDownloads) || defaultParallelDownloads}
          min={1}
          max={8}
          onCommit={(n) => SetParallelDownloads(n)}
        />
      </SettingRow>
      <SettingRow
        label={t`Default profile for Nexus links`}
        description={t`Where nxm downloads go. Empty follows the last opened profile.`}
      >
        <PrefSelect
          value={useSettings((s) => gamePrefs(s).nxmDefaultProfile) || ''}
          onChange={(v) => persist(() => SetNxmDefaultProfile(v), push, fail)}
          options={[
            { value: '', label: t`Last opened profile` },
            ...(profiles ?? [])
              .filter((p) => !p.hidden)
              .map((p) => ({ value: p.id, label: p.name })),
          ]}
        />
      </SettingRow>
    </>
  )
}
