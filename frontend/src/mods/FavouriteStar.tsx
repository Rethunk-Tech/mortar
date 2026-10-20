import { useLingui } from '@lingui/react/macro'
import { IconButton } from '@mui/material'
import { Star } from 'lucide-react'
import { pictureMarkSx } from '../brand/sources/sourceIcons.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { setStarred, useStarred } from './favourites.ts'

const SIZE = 15
const ON_PICTURE_SIZE = 12

// The star on a mod: filled when the game stars the mod, a click toggles it. It is for a mod with a key (a site).
// onPicture lays it over the top-left corner of the mod's picture.
export function FavouriteStar({
  starKey,
  onPicture = false,
}: {
  starKey: string
  onPicture?: boolean
}) {
  const { t } = useLingui()
  const game = useCurrentGame()
  const on = useStarred(game, starKey)
  if (starKey === '') {
    return null
  }
  return (
    <IconButton
      size="small"
      aria-pressed={on}
      aria-label={on ? t`Remove from favourites` : t`Add to favourites`}
      onClick={(e) => {
        e.stopPropagation()
        setStarred(game, starKey, !on)
      }}
      sx={{
        p: 0.25,
        color: on ? 'warning.main' : 'text.disabled',
        ...(onPicture
          ? {
              ...pictureMarkSx,
              position: 'absolute',
              top: 2,
              left: 2,
              // Above a card title's click-anywhere overlay.
              zIndex: 2,
              p: '2px',
              '&:hover': { bgcolor: 'var(--mortar-raised)', color: 'warning.main' },
            }
          : {}),
      }}
    >
      <Star
        size={onPicture ? ON_PICTURE_SIZE : SIZE}
        fill={on ? 'currentColor' : 'none'}
        aria-hidden={true}
      />
    </IconButton>
  )
}
