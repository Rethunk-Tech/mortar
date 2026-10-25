import { useLingui } from '@lingui/react/macro'
import { PROTON, STEAM } from '../brand/productNames.ts'
import { PrefSelect } from './PrefControls.tsx'
import { ScopeNote } from './PrefRow.tsx'
import { persist } from './persist.ts'
import { useProtonBuilds } from './protonBuilds.ts'
import { SettingRow } from './SettingsSection.tsx'
import { useScopedKey } from './scopedKey.ts'
import { useSettings } from './store.ts'

// Which of Steam's Proton builds a Windows copy outside Steam runs through; the builds are this computer's own.
export function ProtonRow({ game }: { game: string }) {
  const { t } = useLingui()
  const builds = useProtonBuilds(game)
  const scoped = useScopedKey(
    'protonVersion',
    game,
    useSettings((s) => s.games?.[game]?.protonVersion ?? ''),
  )
  if (builds.length === 0) {
    return null
  }
  const label = t`${{ name: PROTON }} version`
  const options = [
    { value: '', label: t`The newest ${{ launcher: STEAM }} has` },
    ...builds.map((b) => ({ value: b, label: b })),
  ]
  return (
    <SettingRow
      label={label}
      description={t`The ${{ app: PROTON }} build from ${{ launcher: STEAM }} that runs this copy, which ${{ launcher: STEAM }} does not start`}
    >
      <PrefSelect
        value={scoped.value}
        options={options}
        label={label}
        onChange={(v) => persist(() => scoped.save(v), t`Could not save that setting`)}
      />
      {scoped.profileScope ? (
        <ScopeNote
          own={scoped.own}
          gameLabel={options.find((o) => o.value === scoped.gameValue)?.label ?? scoped.gameValue}
          onReset={() => persist(scoped.reset, t`Could not save that setting`)}
        />
      ) : null}
    </SettingRow>
  )
}
