import { useLingui } from '@lingui/react/macro'
import { useLoaded } from '@rethunk/react-load-hooks'
import { RefreshCcw } from 'lucide-react'
import type { EverywherePreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { PreviewEverywhere } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { ICON_SIZE } from './menu.ts'

export function EverywhereMenuItem({
  game,
  id,
  onOpen,
}: {
  game: string
  id: string
  onOpen: () => void
}) {
  const { t } = useLingui()
  // A preview that cannot be read leaves the action without its count.
  const { data: preview } = useLoaded<EverywherePreview | null>(
    game === '' || id === '' ? null : () => PreviewEverywhere(game, id),
    [game, id],
    null,
  )
  const n = preview?.affected?.length ?? 0
  return (
    <MenuAction
      icon={<RefreshCcw size={ICON_SIZE} />}
      label={t`Update in all profiles (${{ count: n }})`}
      onClick={onOpen}
    />
  )
}
