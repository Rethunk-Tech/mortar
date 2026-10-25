import { Box } from '@mui/material'
import { memo, useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { hasSourceLogo, PICTURE_MARK_PX, pictureMarkSx } from '../brand/sources/sourceIcons.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { localId } from './dependents.ts'
import { FavouriteStar } from './FavouriteStar.tsx'
import { NewDot } from './NewSince.tsx'

const HASH_MULTIPLIER = 31
const UINT32_BITS = 32
const UINT32_RANGE = 2 ** UINT32_BITS
const HUE_DEGREES = 360
const TILE_FONT_RATIO = 0.5
const DEFAULT_TILE_SIZE = 48
const SMALL_TILE = 32

function hash(s: string): number {
  let h = 0
  for (const c of s) {
    h = (h * HASH_MULTIPLIER + (c.codePointAt(0) ?? 0)) % UINT32_RANGE
  }
  return h
}

/** Where a mod picture loads from: a data URL is the picture itself (a development project's own icon); anything else
 * goes through Mortar's picture cache. */
const pictureSrc = (picture: string) =>
  picture.startsWith('data:') ? picture : `/mod-picture/?u=${encodeURIComponent(picture)}`

// source and starKey lay the mod's source mark and its favourite star over the picture's corners. Memoised: a card
// re-renders when its row's size or page details arrive, and the picture does not change with them.
export const LetterTile = memo(function LetterTileView({
  mod,
  size = DEFAULT_TILE_SIZE,
  fresh = false,
  source = '',
  starKey = '',
}: {
  mod: Pick<Mod, 'id' | 'name'> & { picture?: string }
  size?: number
  fresh?: boolean
  source?: string
  starKey?: string
}) {
  const [failed, setFailed] = useState<string | null>(null)
  const picture = mod.picture && failed !== mod.picture ? mod.picture : ''
  return (
    <Box sx={{ position: 'relative', width: size, height: size, flexShrink: 0 }}>
      <Box
        aria-hidden={true}
        className="tile"
        sx={{
          width: size,
          height: size,
          borderRadius: size < SMALL_TILE ? '4px' : '6px',
          display: 'grid',
          placeItems: 'center',
          overflow: 'hidden',
          fontWeight: 700,
          fontSize: size * TILE_FONT_RATIO,
          // The colour is the letter's; a picture keeps its own transparency over the card, as in Browse.
          bgcolor: picture
            ? 'transparent'
            : `hsl(${hash(localId(mod.id).toLowerCase()) % HUE_DEGREES} 35% 38% / 0.85)`,
        }}
      >
        {picture ? (
          <Box
            component="img"
            alt=""
            loading="lazy"
            referrerPolicy="no-referrer"
            src={pictureSrc(picture)}
            onError={() => setFailed(picture)}
            sx={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        ) : (
          (Array.from(mod.name)[0] ?? '?').toUpperCase()
        )}
      </Box>
      <NewDot show={fresh} />
      {hasSourceLogo(source) ? (
        <Box
          role="img"
          aria-label={sourceLabel(source)}
          title={sourceLabel(source)}
          sx={{ ...pictureMarkSx, position: 'absolute', top: 2, right: 2 }}
        >
          <SourceLogo id={source} size={PICTURE_MARK_PX} />
        </Box>
      ) : null}
      {starKey ? <FavouriteStar starKey={starKey} onPicture={true} /> : null}
    </Box>
  )
})
