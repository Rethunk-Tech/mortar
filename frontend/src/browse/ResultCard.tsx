import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Card, Typography } from '@mui/material'
import { ExternalLink } from 'lucide-react'
import { type MouseEvent, useState } from 'react'
import { formatAuthors } from '../mods/authorNormalize.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { GRAY_OPACITY, openPageLabel, PICTURE_PX, ROW_PICTURE_PX } from './browseConstants.ts'

import type { BrowseModes } from './browseModes.ts'
import type { BrowseItem, ResultCardProps } from './browseTypes.ts'
import { pictureSrc } from './pictureSrc.ts'
import { SourceBadges } from './SourceBadges.tsx'
import { hitKey, useBrowseSelection } from './selection.ts'
import { useStats } from './stats.ts'
import { useCardAction } from './useCardAction.tsx'

function CardPicture({ picture, size, dim }: { picture: string; size: number; dim: number }) {
  const sx = { width: size, height: size, flexShrink: 0, borderRadius: '4px', opacity: dim }
  const [failed, setFailed] = useState(false)
  if (picture === '' || failed) {
    return <Box sx={{ ...sx, bgcolor: 'var(--mortar-raised)' }} />
  }
  return (
    <Box
      component="img"
      src={pictureSrc(picture)}
      alt=""
      loading="lazy"
      onError={() => setFailed(true)}
      sx={{ ...sx, objectFit: 'cover' }}
    />
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

// A click on the card's own buttons and links does what they say, not select the card.
const onControl = (e: MouseEvent<HTMLElement>) =>
  e.target instanceof Element && e.target.closest('button, a, [role="button"]') !== null

function ResultCard(props: ResultCardProps) {
  const { row, item, openUrl, modes, sourceNames } = props
  const { t } = useLingui()
  const { name, summary, author } = item
  // The same mod found on several sources is one card; its first source is the default and a badge picks another.
  const sources = [item.source, ...(item.alts ?? []).map((a) => a.source)]
  const [picked, setPicked] = useState(item.source)
  const source = sources.includes(picked) ? picked : item.source
  const { shownItem, url, installed, action } = useCardAction(props, item, source)
  const stats = useStats(shownItem)
  const selected = useBrowseSelection(
    (s) => s.selected !== null && hitKey(s.selected.item) === hitKey(item),
  )
  const dim = isGray(modes, shownItem, installed) ? GRAY_OPACITY : 1
  return (
    <Card
      data-hit={hitKey(item)}
      aria-current={selected ? 'true' : undefined}
      onClick={(e) => {
        if (!onControl(e)) {
          useBrowseSelection.getState().select({ item, source })
        }
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
      <CardPicture picture={item.picture} size={row ? ROW_PICTURE_PX : PICTURE_PX} dim={dim} />
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
        {/* The title is the card's keyboard stop: Enter opens the details, the menu key the card's menu. */}
        <ButtonBase
          aria-label={t`Details of ${item.name}`}
          onClick={() => useBrowseSelection.getState().select({ item, source })}
          sx={{ display: 'block', minWidth: 0, textAlign: 'left', fontFamily: 'inherit' }}
        >
          <Typography noWrap={true} title={name} sx={{ fontSize: 15, fontWeight: 600 }}>
            {name}
          </Typography>
        </ButtonBase>
        <Typography noWrap={true} sx={{ fontSize: 12, color: 'text.secondary' }}>
          {author === '' ? stats : `${formatAuthors(author)} · ${stats}`}
        </Typography>
        <Typography
          title={summary}
          sx={{
            fontSize: 13,
            color: 'var(--mortar-ink-soft)',
            display: '-webkit-box',
            WebkitLineClamp: row ? 1 : 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
          }}
        >
          {summary}
        </Typography>
        <SourceBadges sources={sources} picked={source} names={sourceNames} onPick={setPicked} />
      </Box>
      <Box
        sx={{
          display: 'flex',
          flexDirection: row ? 'row' : 'column',
          alignItems: row ? 'center' : 'flex-end',
          justifyContent: 'space-between',
          gap: 0.5,
        }}
      >
        <IconAction
          label={openPageLabel(source)}
          icon={<ExternalLink size={15} />}
          onClick={() => openUrl(url)}
        />
        {action}
      </Box>
    </Card>
  )
}

export { CardPicture, ResultCard }
