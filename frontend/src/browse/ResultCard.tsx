import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Card, Typography } from '@mui/material'
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
  const sx = {
    width: size,
    height: size,
    flexShrink: 0,
    borderRadius: '4px',
    opacity: dim,
  } as const
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
  const logo = source && hasSourceLogo(source.id) ? source : undefined
  if (!(logo || starKey)) {
    return image
  }
  const mark = PICTURE_MARK_PX
  return (
    <Box
      sx={{
        position: 'relative',
        flexShrink: 0,
        display: 'flex',
      }}
    >
      {image}
      {logo && pick && pick.sources.length > 1 ? (
        <SourcePicks pick={pick} chosen={logo.id} mark={mark} />
      ) : null}
      {logo && !(pick && pick.sources.length > 1) ? (
        <Box
          role="img"
          aria-label={logo.name}
          title={logo.name}
          sx={{ ...corner, ...pictureMarkSx }}
        >
          <SourceLogo id={logo.id} size={mark} />
        </Box>
      ) : null}
      {starKey ? <FavouriteStar starKey={starKey} onPicture={true} /> : null}
    </Box>
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
    <Box sx={{ ...corner, display: 'flex', flexDirection: 'column', gap: '2px' }}>
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
  '& .MuiChip-icon': { display: 'none' },
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
        <Box
          component="span"
          title={flagText}
          sx={{ display: 'inline-flex', flexShrink: 0, color: `${flag}.main` }}
        >
          <TriangleAlert size={15} aria-hidden={true} />
        </Box>
      ) : null}
      <Typography noWrap={true} title={name} sx={{ fontSize: 'lead.fontSize', fontWeight: 600 }}>
        {name}
      </Typography>
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
  return (
    <Typography
      noWrap={true}
      sx={{ fontSize: 'small.fontSize', fontWeight: 500, color: 'var(--mortar-accent-ink)' }}
    >
      {formatAuthors(author)}
    </Typography>
  )
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
      <Box
        component="span"
        role="img"
        aria-label={note}
        sx={{ display: 'inline-flex', flexShrink: 0, ml: 'auto', color: 'warning.main' }}
      >
        <TriangleAlert size={15} aria-hidden={true} />
      </Box>
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
    <Box
      sx={{
        mt: 'auto',
        alignSelf: 'flex-end',
        display: 'flex',
        flexWrap: 'wrap',
        justifyContent: 'flex-end',
        columnGap: 1,
        fontSize: 'small.fontSize',
        color: 'var(--mortar-ink-dim-60)',
        fontVariantNumeric: 'tabular-nums',
      }}
    >
      {stats.map(({ id, icon, short, full }) => {
        const Icon = STAT_ICONS[icon]
        return (
          <Box
            component="span"
            key={id}
            role="img"
            aria-label={full}
            title={full}
            sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.375, whiteSpace: 'nowrap' }}
          >
            <Icon size={12} aria-hidden={true} />
            {short}
          </Box>
        )
      })}
    </Box>
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
        <CardPicture
          picture={item.picture}
          size={PICTURE_PX}
          dim={dim}
          source={mark}
          starKey={favouriteKey(item.source, item.id)}
        />
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
          <CardPicture
            picture={item.picture}
            size={PICTURE_PX}
            dim={dim}
            source={mark}
            pick={{ sources, names: sourceNames, onPick: setPicked }}
            starKey={favouriteKey(item.source, item.id)}
          />
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
          <ReuploadMark reuploadOf={item.reuploadOf ?? ''} />
        </Box>
        <Byline author={author} />
        <Typography
          title={summary}
          sx={{
            fontSize: 'base.fontSize',
            color: 'var(--mortar-ink-85)',
            display: '-webkit-box',
            WebkitLineClamp: row ? 1 : 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
          }}
        >
          {summary}
        </Typography>
        <StatsLine stats={stats} />
        {row ? (
          <SourceBadges sources={sources} picked={source} names={sourceNames} onPick={setPicked} />
        ) : null}
      </Box>
      {row ? action : null}
    </Card>
  )
})

export { CardPicture, ResultCard }
