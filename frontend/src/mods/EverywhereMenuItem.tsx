import { useLingui } from '@lingui/react/macro'
import { MenuItem } from '@mui/material'
import { useEffect, useState } from 'react'
import type { EverywherePreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { PreviewEverywhere } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
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
  const [preview, setPreview] = useState<EverywherePreview | null>(null)
  useEffect(() => {
    if (game === '' || id === '') {
      return
    }
    PreviewEverywhere(game, id)
      .then(setPreview)
      .catch(() => setPreview(null))
  }, [game, id])
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
        mods={[{ id, newKey: 'latest' }]}
        onClose={() => setOpen(false)}
      />
    </>
  )
}
