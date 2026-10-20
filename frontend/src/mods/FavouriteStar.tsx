import { useLingui } from '@lingui/react/macro'
import { IconButton } from '@mui/material'
import { Star } from 'lucide-react'
import { PICTURE_MARK_PX } from '../brand/sources/sourceIcons.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { setStarred, useStarred } from './favourites.ts'

const SIZE = 15

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
              position: 'absolute',
              top: 2,
              left: 2,
              p: '2px',
              // Above a card title's click-anywhere overlay.
              zIndex: 2,
              // No chip behind it: a dark edge keeps the outline readable on any art.
              color: on ? 'warning.main' : 'common.white',
              filter:
                'drop-shadow(0 0 1.5px rgba(0, 0, 0, 0.95)) drop-shadow(0 1px 2px rgba(0, 0, 0, 0.7))',
              '&:hover': { bgcolor: 'transparent', color: 'warning.main' },
            }
          : {}),
      }}
    >
      <Star
        size={onPicture ? PICTURE_MARK_PX : SIZE}
        fill={on ? 'currentColor' : 'none'}
        aria-hidden={true}
      />
    </IconButton>
  )
}
