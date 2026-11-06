import { useLingui } from '@lingui/react/macro'
import { RefreshCcw } from 'lucide-react'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { ICON_SIZE } from './menu.ts'
import { useEverywhere } from './useEverywhere.ts'

export function EverywhereMenuItem({
  game,
  id,
  update,
  onOpen,
}: {
  game: string
  id: string
  update: { source: string; nexusId: number }
  onOpen: () => void
}) {
  const { t } = useLingui()
  const { count, why } = useEverywhere(game, id, update)
  return (
    <DisabledReason title={why} disabled={why !== ''}>
      <MenuAction
        icon={<RefreshCcw size={ICON_SIZE} />}
        label={t`Update in all profiles (${{ count }})`}
        disabled={why !== ''}
        onClick={onOpen}
      />
    </DisabledReason>
  )
}
