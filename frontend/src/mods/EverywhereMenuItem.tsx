import { useLingui } from '@lingui/react/macro'
import { MenuItem } from '@mui/material'
import { useEffect, useState } from 'react'
import type { EverywherePreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { PreviewEverywhere } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { EverywhereDialog } from './updateReview/EverywhereDialog.tsx'

export function EverywhereMenuItem({
  game,
  uniqueId,
  close,
}: {
  game: string
  uniqueId: string
  close: () => void
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const [preview, setPreview] = useState<EverywherePreview | null>(null)
  useEffect(() => {
    if (game === '' || uniqueId === '') {
      return
    }
    PreviewEverywhere(game, uniqueId)
      .then(setPreview)
      .catch(() => setPreview(null))
  }, [game, uniqueId])
  const n = preview?.affected?.length ?? 0
  return (
    <>
      <MenuItem
        onClick={() => {
          close()
          setOpen(true)
        }}
      >
        {t`Update in all profiles (${n})`}
      </MenuItem>
      <EverywhereDialog
        open={open}
        game={game}
        mods={[{ id: uniqueId, newKey: 'latest' }]}
        onClose={() => setOpen(false)}
      />
    </>
  )
}
