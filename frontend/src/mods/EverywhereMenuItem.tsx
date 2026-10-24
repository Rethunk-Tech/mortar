import { useLingui } from '@lingui/react/macro'
import { RefreshCcw } from 'lucide-react'
import { useState } from 'react'
import type { EverywherePreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { PreviewEverywhere } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { ICON_SIZE } from './menu.ts'
import { EverywhereDialog } from './updateReview/EverywhereDialog.tsx'

export function EverywhereMenuItem({
  game,
  id,
  close,
}: {
  game: string
  id: string
  close: () => void
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  // A preview that cannot be read leaves the action without its count.
  const { data: preview } = useLoaded<EverywherePreview | null>(
    game === '' || id === '' ? null : () => PreviewEverywhere(game, id),
    [game, id],
    null,
  )
  const n = preview?.affected?.length ?? 0
  return (
    <>
      <MenuAction
        icon={<RefreshCcw size={ICON_SIZE} />}
        label={t`Update in all profiles (${{ count: n }})`}
        onClick={() => {
          close()
          setOpen(true)
        }}
      />
      <EverywhereDialog
        open={open}
        game={game}
        mods={[{ id, newKey: 'latest' }]}
        onClose={() => setOpen(false)}
      />
    </>
  )
}
