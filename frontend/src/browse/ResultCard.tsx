import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Card, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import { type MouseEvent, memo, useState } from 'react'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { hasSourceLogo } from '../brand/sources/sourceIcons.ts'
import { formatAuthors } from '../mods/authorNormalize.ts'
import { FavouriteStar } from '../mods/FavouriteStar.tsx'
import { favouriteKey } from '../mods/favourites.ts'
import { MIDDLE, opensPage } from '../mods/pageClick.ts'
import { useSettings } from '../settings/store.ts'
import { BUTTON_RADIUS_PX, BUTTON_SMALL_HEIGHT } from '../theme/theme.ts'
import { GRAY_OPACITY, PICTURE_PX, ROW_PICTURE_PX } from './browseConstants.ts'
import type { BrowseModes } from './browseModes.ts'
import type { BrowseItem, ResultCardProps } from './browseTypes.ts'
import { pictureSrc } from './pictureSrc.ts'
import { SourceBadges } from './SourceBadges.tsx'
import { hitKey, useBrowseSelection } from './selection.ts'
import { parseCardStats, useCardStats } from './stats.ts'
import { useCardAction } from './useCardAction.tsx'

// The source mark in the picture's corner is about a third of the picture, within what stays legible.
const MARK_SHARE = 0.3
const MARK_MIN_PX = 12
const MARK_MAX_PX = 16

// A mod's picture, with the mark of the site Add installs from in its corner, so every card says where it comes from.
function CardPicture({
  picture,
  size,
  dim,
  source,
}: {
  picture: string
  size: number
  dim: number
  source?: { id: string; name: string }
}) {
  const sx = { width: size, height: size, flexShrink: 0, borderRadius: '4px', opacity: dim }
  const [failed, setFailed] = useState(false)
  const image =
    picture === '' || failed ? (
      <Box sx={{ ...sx, bgcolor: 'var(--mortar-raised)' }} />
    ) : (
      <Box
        component="img"
        src={pictureSrc(picture)}
        alt=""
        loading="lazy"
        onError={() => setFailed(true)}
        sx={{ ...sx, objectFit: 'cover' }}
      />
    )
  if (!(source && hasSourceLogo(source.id))) {
    return image
  }
  const mark = Math.min(MARK_MAX_PX, Math.max(MARK_MIN_PX, Math.round(size * MARK_SHARE)))
  return (
    <Box sx={{ position: 'relative', flexShrink: 0, display: 'flex' }}>
      {image}
      <Box
        role="img"
        aria-label={source.name}
        title={source.name}
        sx={{
          position: 'absolute',
          top: 2,
          right: 2,
          display: 'grid',
          placeItems: 'center',
          p: '1px',
          borderRadius: '4px',
          bgcolor: 'var(--mortar-raised)',
          border: '1px solid var(--mortar-hairline)',
        }}
      >
        <SourceLogo id={source.id} size={mark} />
      </Box>
    </Box>
  )
}

// Gray out dims the mod's picture and text; the action and its chip stay readable.
function isGray(modes: BrowseModes, item: BrowseItem, installed: boolean): boolean {
  return (
    (modes.installed === 'gray' && installed) ||
    (modes.obsolete === 'gray' && item.obsolete) ||
    (modes.broken === 'gray' && item.broken)
  )
}

// Flag marks a broken mod in the error colour and an obsolete one in the warning colour; broken wins when both apply.
function flagOf(modes: BrowseModes, item: BrowseItem): 'error' | 'warning' | null {
  if (modes.broken === 'flag' && item.broken) {
    return 'error'
  }
  return modes.obsolete === 'flag' && item.obsolete ? 'warning' : null
}

// Under a grid card's picture the action is as wide as the picture: a button's label wraps rather than widening the
// column, and drops its icon to fit.
const underPictureSx = {
  '& .MuiButton-root': {
    width: '100%',
    minWidth: 0,
    height: 'auto',
    minHeight: BUTTON_SMALL_HEIGHT,
    px: 0.5,
    py: 0.5,
    whiteSpace: 'normal',
    lineHeight: 1.2,
  },
  '& .MuiButton-startIcon': { display: 'none' },
  '& .card-progress': { width: '100%' },
  // A status chip (Added, Loader) takes the button's place and shape.
  '& .MuiChip-root': {
    width: '100%',
    height: BUTTON_SMALL_HEIGHT,
    borderRadius: `${BUTTON_RADIUS_PX}px`,
  },
} as const

// The title is the card's keyboard stop: Enter opens the details, the menu key the card's menu. A flagged mod's
// warning leads it and its accessible name.
function CardTitle(props: { name: string; flag: 'error' | 'warning' | null; onOpen: () => void }) {
  const { name, flag, onOpen } = props
  const { t } = useLingui()
  const flagText = flag === 'error' ? t`Marked broken` : t`Deprecated or obsolete`
  const details = t`Details of ${props.name}`
  return (
    <ButtonBase
      aria-label={flag ? `${flagText}. ${details}` : details}
      onClick={onOpen}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-start',
        gap: 0.5,
        minWidth: 0,
        textAlign: 'left',
        fontFamily: 'inherit',
      }}
    >
      {flag ? (
        <Box
          component="span"
          title={flagText}
          sx={{ display: 'inline-flex', flexShrink: 0, color: `${flag}.main` }}
        >
          <TriangleAlert size={15} aria-hidden={true} />
        </Box>
      ) : null}
      <Typography noWrap={true} title={name} sx={{ fontSize: 15, fontWeight: 600 }}>
        {name}
      </Typography>
    </ButtonBase>
  )
}

// A click on the card's own buttons and links does what they say, not select the card.
const onControl = (e: MouseEvent<HTMLElement>) =>
  e.target instanceof Element && e.target.closest('button, a, [role="button"]') !== null

const ResultCard = memo(function BrowseResultCard(props: ResultCardProps) {
  const { row, item, modes, sourceNames } = props
  const { name, summary, author } = item
  // The same mod found on several sources is one card; its first source is the default and a badge picks another.
  const sources = [item.source, ...(item.alts ?? []).map((a) => a.source)]
  const [picked, setPicked] = useState(item.source)
  const source = sources.includes(picked) ? picked : item.source
  const mark = { id: source, name: sourceNames.get(source) ?? source }
  const { shownItem, installed, action } = useCardAction(props, item, source)
  const shownStats = parseCardStats(useSettings((s) => s.browseCardStats))
  const stats = useCardStats(shownItem, shownStats)
  const selected = useBrowseSelection(
    (s) => s.selected !== null && hitKey(s.selected.item) === hitKey(item),
  )
  const dim = isGray(modes, shownItem, installed) ? GRAY_OPACITY : 1
  const flag = flagOf(modes, shownItem)
  return (
    <Card
      data-hit={hitKey(item)}
      aria-current={selected ? 'true' : undefined}
      onMouseDown={(e) => {
        if (e.button === MIDDLE) {
          e.preventDefault()
        }
      }}
      onAuxClick={(e) => {
        if (opensPage(e, true) && !onControl(e)) {
          props.openUrl(shownItem.url)
        }
      }}
      onClick={(e) => {
        if (onControl(e)) {
          return
        }
        if (opensPage(e, true)) {
          props.openUrl(shownItem.url)
          return
        }
        useBrowseSelection.getState().select({ item, source })
      }}
      onContextMenu={(e) => {
        e.preventDefault()
        useBrowseSelection
          .getState()
          .openMenu({ item, source }, { top: e.clientY, left: e.clientX })
      }}
      sx={{
        display: 'flex',
        alignItems: row ? 'center' : 'stretch',
        gap: 1.25,
        p: 1,
        borderRadius: '6px',
        minWidth: 0,
        cursor: 'pointer',
        outline: selected ? '2px solid' : 'none',
        outlineColor: 'primary.main',
        outlineOffset: '-2px',
      }}
    >
      {row ? (
        <CardPicture picture={item.picture} size={ROW_PICTURE_PX} dim={dim} source={mark} />
      ) : (
        <Box
          sx={{
            width: PICTURE_PX,
            flexShrink: 0,
            display: 'flex',
            flexDirection: 'column',
            gap: 0.75,
            ...underPictureSx,
          }}
        >
          <CardPicture picture={item.picture} size={PICTURE_PX} dim={dim} source={mark} />
          {action}
        </Box>
      )}
      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 0.25,
          opacity: dim,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, minWidth: 0 }}>
          <CardTitle
            name={name}
            flag={flag}
            onOpen={() => useBrowseSelection.getState().select({ item, source })}
          />
          <FavouriteStar starKey={favouriteKey(item.source, item.id)} />
        </Box>
        {author === '' ? null : (
          <Typography
            noWrap={true}
            sx={{ fontSize: 12, fontWeight: 500, color: 'var(--mortar-accent-ink)' }}
          >
            {formatAuthors(author)}
          </Typography>
        )}
        <Typography
          title={summary}
          sx={{
            fontSize: 13,
            color: 'var(--mortar-ink-85)',
            display: '-webkit-box',
            WebkitLineClamp: row ? 1 : 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
          }}
        >
          {summary}
        </Typography>
        {stats.length === 0 ? null : (
          // Each stat stays whole and the line wraps between them on a narrow card.
          <Typography
            sx={{
              fontSize: 12,
              color: 'var(--mortar-ink-dim-60)',
              fontVariantNumeric: 'tabular-nums',
              display: 'flex',
              flexWrap: 'wrap',
              columnGap: 0.75,
            }}
          >
            {stats.map((s, i) => (
              <Box component="span" key={s} sx={{ whiteSpace: 'nowrap' }}>
                {i === 0 ? s : `· ${s}`}
              </Box>
            ))}
          </Typography>
        )}
        <SourceBadges sources={sources} picked={source} names={sourceNames} onPick={setPicked} />
      </Box>
      {row ? action : null}
    </Card>
  )
})

export { CardPicture, ResultCard }
