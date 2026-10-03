import { useLingui } from '@lingui/react/macro'
import { SetNxmDefaultProfile } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSelect } from '../PrefControls.tsx'
import { PrefByKey } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { prefCopy } from '../prefCopy.ts'
import { specByKey, usePrefSpecs } from '../prefSpecs.ts'
import { GAME_STARDEW, prefAsString, prefRaw } from '../prefValue.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

export function NexusDownloadPrefs() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  const profiles = useProfiles((s) => s.profiles)
  const specs = usePrefSpecs()
  const spec = specByKey(specs, 'nxmDefaultProfile')
  const settings = useSettings()
  const copy = prefCopy(t, 'nxmDefaultProfile')
  const value = spec
    ? prefAsString(prefRaw(settings, spec, GAME_STARDEW), spec)
    : (settings.games?.[GAME_STARDEW]?.nxmDefaultProfile ?? '')
  return (
    <>
      <PrefByKey prefKey="autoTrackNexus" />
      <PrefByKey prefKey="parallelDownloads" />
      <SettingRow label={copy.label} description={copy.description}>
        <PrefSelect
          value={value}
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
