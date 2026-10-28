import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Card } from '@mui/material'
import {
  Clock,
  Download,
  HardDrive,
  Heart,
  type LucideIcon,
  Star,
  Tag,
  ThumbsUp,
  TriangleAlert,
  Users,
} from 'lucide-react'
import { type MouseEvent, memo, useState } from 'react'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { hasSourceLogo, PICTURE_MARK_PX, pictureMarkSx } from '../brand/sources/sourceIcons.ts'
import { formatAuthors } from '../mods/authorNormalize.ts'
import { FavouriteStar } from '../mods/FavouriteStar.tsx'
import { favouriteKey } from '../mods/favourites.ts'
import { MIDDLE, opensPage } from '../mods/pageClick.ts'
import { useSettings } from '../settings/store.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { space } from '../theme/density.ts'
import { BUTTON_RADIUS_PX, BUTTON_SMALL_HEIGHT } from '../theme/theme.ts'
import { GRAY_OPACITY, PICTURE_PX, STALE_OPACITY } from './browseConstants.ts'
import type { BrowseModes } from './browseModes.ts'
import type { BrowseItem, ResultCardProps } from './browseTypes.ts'
import { pictureSrc } from './pictureSrc.ts'
import { SourceBadges } from './SourceBadges.tsx'
import { hitKey, useBrowseSelection } from './selection.ts'
import { type CardStatPart, parseCardStats, type StatIcon, useCardStats } from './stats.ts'
import { useCardAction } from './useCardAction.tsx'

// The source mark in the picture's corner is about a third of the picture, within what stays legible.

const corner = { position: 'absolute', top: 2, right: 2 } as const

// A mod's picture, with the mark of the site Add installs from in its corner, so every card says where it comes from.
// A mod on several sites stacks one mark per site there instead, and a click picks the site Add installs from.
function CardPicture({
  picture,
  size,
  dim,
  source,
  pick,
  starKey,
}: {
  picture: string
  size: number
  dim: number
  source?: { id: string; name: string }
  pick?: { sources: string[]; names: Map<string, string>; onPick: (source: string) => void }
  starKey?: string
}) {
  // Inline styles rather than Box: a results page draws one picture per card, and each Box is three components.
  const style = {
    width: size,
    height: size,
    flexShrink: 0,
    borderRadius: '4px',
    opacity: dim,
  } as const
  const [failed, setFailed] = useState(false)
  const image =
    picture === '' || failed ? (
      <span style={{ ...style, display: 'block', backgroundColor: 'var(--mortar-raised)' }} />
    ) : (
      // A Box: a plain img may not take the onError that swaps in the blank tile.
      <Box
        component="img"
        src={pictureSrc(picture)}
        alt=""
        loading="lazy"
        onError={() => setFailed(true)}
        style={{ ...style, objectFit: 'cover' }}
      />
    )
  const logo = source && hasSourceLogo(source.id) ? source : undefined
  if (!(logo || starKey)) {
    return image
  }
  const mark = PICTURE_MARK_PX
  return (
    <div style={{ position: 'relative', flexShrink: 0, display: 'flex' }}>
      {image}
      {logo && pick && pick.sources.length > 1 ? (
        <SourcePicks pick={pick} chosen={logo.id} mark={mark} />
      ) : null}
      {logo && !(pick && pick.sources.length > 1) ? (
        <div
          role="img"
          aria-label={logo.name}
          title={logo.name}
          style={{ ...corner, ...pictureMarkSx }}
        >
          <SourceLogo id={logo.id} size={mark} />
        </div>
      ) : null}
      {starKey ? <FavouriteStar starKey={starKey} onPicture={true} /> : null}
    </div>
  )
}

function SourcePicks({
  pick,
  chosen,
  mark,
}: {
  pick: { sources: string[]; names: Map<string, string>; onPick: (source: string) => void }
  chosen: string
  mark: number
}) {
  return (
    <div style={{ ...corner, display: 'flex', flexDirection: 'column', gap: '2px' }}>
      {pick.sources.map((id) => {
        const name = pick.names.get(id) ?? id
        const on = id === chosen
        return (
          <Tooltip key={id} title={name}>
            <ButtonBase
              aria-label={name}
              aria-pressed={on}
              onClick={() => pick.onPick(id)}
              sx={{
                ...pictureMarkSx,
                opacity: on ? 1 : STALE_OPACITY,
                borderColor: on ? 'var(--mortar-ink-dim-60)' : 'var(--mortar-hairline)',
                '&:hover': { opacity: 1 },
              }}
            >
              {hasSourceLogo(id) ? <SourceLogo id={id} size={mark} /> : name.slice(0, 1)}
            </ButtonBase>
          </Tooltip>
        )
      })}
    </div>
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
  '& .MuiChip-icon': { display: 'none' },
} as const

// The card's parts are plain elements styled from the card: a results page redraws every card at once (a view switch,
// a page), and each Box or Typography is three components more per part.
const cardSx = {
  display: 'flex',
  gap: 1.25,
  p: 1,
  borderRadius: '6px',
  minWidth: 0,
  cursor: 'pointer',
  outlineOffset: '-2px',
  '& .card-body': { flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 0.25 },
  '& .card-head': { display: 'flex', alignItems: 'center', gap: 0.5, minWidth: 0 },
  '& .card-name': {
    // Typography's body1 metrics, which the plain element would not get.
    typography: 'body1',
    display: 'block',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    fontSize: 'lead.fontSize',
    fontWeight: 600,
  },
  '& .card-flag': { display: 'inline-flex', flexShrink: 0 },
  '& .card-flag-error': { color: 'error.main' },
  '& .card-flag-warning': { color: 'warning.main' },
  '& .card-reupload': { display: 'inline-flex', flexShrink: 0, ml: 'auto', color: 'warning.main' },
  '& .card-byline': {
    // Typography's body1 metrics, which the plain element would not get.
    typography: 'body1',
    m: 0,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    fontSize: 'small.fontSize',
    fontWeight: 500,
    color: 'var(--mortar-accent-ink)',
  },
  '& .card-summary': {
    // Typography's body1 metrics, which the plain element would not get.
    typography: 'body1',
    m: 0,
    fontSize: 'base.fontSize',
    color: 'var(--mortar-ink-85)',
    display: '-webkit-box',
    WebkitLineClamp: 2,
    WebkitBoxOrient: 'vertical',
    overflow: 'hidden',
  },
  '& .card-stats': {
    mt: 'auto',
    alignSelf: 'flex-end',
    display: 'flex',
    flexWrap: 'wrap',
    justifyContent: 'flex-end',
    columnGap: space.gap,
    fontSize: 'small.fontSize',
    color: 'var(--mortar-ink-dim-60)',
    fontVariantNumeric: 'tabular-nums',
  },
  '& .card-stat': {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 0.375,
    whiteSpace: 'nowrap',
  },
  '& .card-under': {
    width: PICTURE_PX,
    flexShrink: 0,
    display: 'flex',
    flexDirection: 'column',
    gap: 0.75,
    ...underPictureSx,
  },
  '&.card-row': { alignItems: 'center' },
  '&.card-row .card-summary': { WebkitLineClamp: 1 },
} as const

// The title is the card's keyboard stop: Enter opens the details, the menu key the card's menu. A flagged mod's
// warning leads it and its accessible name.
function CardTitle(props: { name: string; flag: 'error' | 'warning' | null; onOpen: () => void }) {
  const { name, flag, onOpen } = props
  const { t } = useLingui()
  const flagText = flag === 'error' ? t`Marked broken` : t`Deprecated or obsolete`
  const details = t`Details of ${{ name: props.name }}`
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
        <span className={`card-flag card-flag-${flag}`} title={flagText}>
          <TriangleAlert size={15} aria-hidden={true} />
        </span>
      ) : null}
      <span className="card-name" title={name}>
        {name}
      </span>
    </ButtonBase>
  )
}

// A click on the card's own buttons and links does what they say, not select the card.
const onControl = (e: MouseEvent<HTMLElement>) =>
  e.target instanceof Element && e.target.closest('button, a, [role="button"]') !== null

function Byline({ author }: { author: string }) {
  if (author === '') {
    return null
  }
  return <p className="card-byline">{formatAuthors(author)}</p>
}

// When another site lists a mod of this name by someone else with more downloads: a mark at the card's top right, the
// note in its tooltip, so the card keeps its height.
function ReuploadMark({ reuploadOf }: { reuploadOf: string }) {
  const { t } = useLingui()
  if (reuploadOf === '') {
    return null
  }
  const note = t`Possibly a re-upload of ${reuploadOf}`
  return (
    <Tooltip title={note}>
      <span className="card-reupload" role="img" aria-label={note}>
        <TriangleAlert size={15} aria-hidden={true} />
      </span>
    </Tooltip>
  )
}

const STAT_ICONS: Record<StatIcon, LucideIcon> = {
  like: Heart,
  endorsement: ThumbsUp,
  star: Star,
  follower: Users,
  download: Download,
  size: HardDrive,
  version: Tag,
  updated: Clock,
}

// Icons and short figures, at the card's bottom right; each stat's tooltip and accessible name has its whole wording.
function StatsLine({ stats }: { stats: CardStatPart[] }) {
  if (stats.length === 0) {
    return null
  }
  return (
    <div className="card-stats">
      {stats.map(({ id, icon, short, full }) => {
        const Icon = STAT_ICONS[icon]
        return (
          <span key={id} className="card-stat" role="img" aria-label={full} title={full}>
            <Icon size={12} aria-hidden={true} />
            {short}
          </span>
        )
      })}
    </div>
  )
}

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
      className={row ? 'card-row' : undefined}
      sx={[
        cardSx,
        selected ? { outline: '2px solid', outlineColor: 'primary.main' } : { outline: 'none' },
      ]}
    >
      {row ? (
        <CardPicture
          picture={item.picture}
          size={PICTURE_PX}
          dim={dim}
          source={mark}
          starKey={favouriteKey(item.source, item.id)}
        />
      ) : (
        <div className="card-under">
          <CardPicture
            picture={item.picture}
            size={PICTURE_PX}
            dim={dim}
            source={mark}
            pick={{ sources, names: sourceNames, onPick: setPicked }}
            starKey={favouriteKey(item.source, item.id)}
          />
          {action}
        </div>
      )}
      <div className="card-body" style={{ opacity: dim }}>
        <div className="card-head">
          <CardTitle
            name={name}
            flag={flag}
            onOpen={() => useBrowseSelection.getState().select({ item, source })}
          />
          <ReuploadMark reuploadOf={item.reuploadOf ?? ''} />
        </div>
        <Byline author={author} />
        <p className="card-summary" title={summary}>
          {summary}
        </p>
        <StatsLine stats={stats} />
        {row ? (
          <SourceBadges sources={sources} picked={source} names={sourceNames} onPick={setPicked} />
        ) : null}
      </div>
      {row ? action : null}
    </Card>
  )
})

export { CardPicture, ResultCard }
