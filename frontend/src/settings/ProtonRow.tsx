import { useLingui } from '@lingui/react/macro'
import { PROTON, STEAM } from '../brand/productNames.ts'
import { useProtonBuilds } from './protonBuilds.ts'
import { ScopedSelectRow } from './ScopedSelectRow.tsx'
import { useSettings } from './store.ts'
import { useSettingsRelevant } from './useSettingsSearch.ts'

// Which of Steam's Proton builds a Windows copy outside Steam runs through; the builds are this computer's own.
export function ProtonRow({ game }: { game: string }) {
  const { t } = useLingui()
  const label = t`${{ name: PROTON }} version`
  const description = t`The ${{ app: PROTON }} build from ${{ launcher: STEAM }} that runs this copy, which ${{ launcher: STEAM }} does not start`
  const builds = useProtonBuilds(game, useSettingsRelevant(label, description))
  const gameValue = useSettings((s) => s.games?.[game]?.protonVersion ?? '')
  if (builds.length === 0) {
    return null
  }
  return (
    <ScopedSelectRow
      settingKey="protonVersion"
      game={game}
      gameValue={gameValue}
      label={label}
      description={description}
      options={[
        { value: '', label: t`The newest ${{ launcher: STEAM }} has` },
        ...builds.map((b) => ({ value: b, label: b })),
      ]}
    />
  )
}
