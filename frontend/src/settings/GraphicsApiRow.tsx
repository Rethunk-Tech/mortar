import { useLingui } from '@lingui/react/macro'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useGameInfo } from '../games/info.ts'
import { PrefSelect } from './PrefControls.tsx'
import { persist } from './persist.ts'
import { SettingRow } from './SettingsSection.tsx'
import { useSettings } from './store.ts'

// The catalog gives each game its own choices, so this row reads them from the game instead of a fixed list.
export function GraphicsApiRow({ game }: { game: string }) {
  const { t } = useLingui()
  const graphics = useGameInfo(game)?.graphics
  const value = useSettings((s) => s.games?.[game]?.graphicsApi ?? '')
  if (!graphics) {
    return null
  }
  const label = t`Graphics API`
  const options = [
    { value: '', label: t`Ask when I press Play` },
    ...(graphics.choices ?? []).map((c) => ({
      value: c.id,
      label: c.id === graphics.recommended ? t`${c.label} (Recommended)` : c.label,
    })),
  ]
  return (
    <SettingRow label={label} description={graphics.explanation}>
      <PrefSelect
        value={value}
        options={options}
        label={label}
        onChange={(v) =>
          persist(() => SetByKey('graphicsApi', v, game), t`Could not save that setting`)
        }
      />
    </SettingRow>
  )
}
