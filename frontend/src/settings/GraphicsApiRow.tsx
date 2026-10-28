import { useLingui } from '@lingui/react/macro'
import { useGameInfo } from '../games/info.ts'
import { ScopedSelectRow } from './ScopedSelectRow.tsx'
import { useSettings } from './store.ts'

// The catalog gives each game its own choices, so this row reads them from the game instead of a fixed list.
export function GraphicsApiRow({ game }: { game: string }) {
  const { t } = useLingui()
  const graphics = useGameInfo(game)?.graphics
  const gameValue = useSettings((s) => s.games?.[game]?.graphicsApi ?? '')
  if (!graphics) {
    return null
  }
  return (
    <ScopedSelectRow
      settingKey="graphicsApi"
      game={game}
      gameValue={gameValue}
      label={t`Graphics API`}
      description={graphics.explanation}
      options={[
        { value: '', label: t`Ask when I press Play` },
        ...(graphics.choices ?? []).map((c) => ({
          value: c.id,
          label: c.id === graphics.recommended ? t`${{ name: c.label }} (recommended)` : c.label,
        })),
      ]}
    />
  )
}
