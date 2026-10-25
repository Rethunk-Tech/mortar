import { useLingui } from '@lingui/react/macro'
import { useGameInfo } from '../games/info.ts'
import { PrefSelect } from './PrefControls.tsx'
import { ScopeNote } from './PrefRow.tsx'
import { persist } from './persist.ts'
import { SettingRow } from './SettingsSection.tsx'
import { useScopedKey } from './scopedKey.ts'
import { useSettings } from './store.ts'

// The catalog gives each game its own choices, so this row reads them from the game instead of a fixed list.
export function GraphicsApiRow({ game }: { game: string }) {
  const { t } = useLingui()
  const graphics = useGameInfo(game)?.graphics
  const scoped = useScopedKey(
    'graphicsApi',
    game,
    useSettings((s) => s.games?.[game]?.graphicsApi ?? ''),
  )
  if (!graphics) {
    return null
  }
  const label = t`Graphics API`
  const options = [
    { value: '', label: t`Ask when I press Play` },
    ...(graphics.choices ?? []).map((c) => ({
      value: c.id,
      label: c.id === graphics.recommended ? t`${{ name: c.label }} (Recommended)` : c.label,
    })),
  ]
  return (
    <SettingRow label={label} description={graphics.explanation}>
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
