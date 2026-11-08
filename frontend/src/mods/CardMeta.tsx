import { Box, Typography } from '@mui/material'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { StatParts } from '../browse/StatsLine.tsx'
import { CARD_STAT_SX, parseCardStats, useModCardStats } from '../browse/stats.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { AuthorLink } from './AuthorLink.tsx'
import { formatAuthors } from './authorNormalize.ts'
import { useEntrySize } from './listRows.ts'
import { nexusIdOf, sourceKind } from './lookup.ts'
import { nexusEntryOf, useNexusDetails } from './nexusDetails.ts'

const META_FONT_PX = 12

// A card's second line (author and version) and, when Card details are ticked, the line of chosen details under it.
export function CardMeta({
  mod: m,
  profile,
  statsLine,
}: {
  mod: Mod
  profile: Profile
  statsLine: boolean
}) {
  const showAuthor = useSettings((s) => s.showAuthorOnCards) !== false
  // Read here rather than from the card's row, so their arrival re-renders this line and not the whole card.
  const size = useEntrySize(m.key)
  const game = useProfiles((s) => s.game?.id ?? '')
  const details = useNexusDetails(
    (s) => nexusEntryOf(s.byKey, game, nexusIdOf(profile, m))?.details,
  )
  const cardStats = useModCardStats(
    {
      mod: m,
      source: sourceKind(profile, m),
      ...(size === undefined ? {} : { size }),
      ...(details ? { details } : {}),
    },
    parseCardStats(useSettings((s) => s.browseCardStats)),
  )
  const byline = [formatAuthors(m.author), m.version].filter(Boolean).join(' · ')
  return (
    <>
      <Typography
        noWrap={true}
        title={showAuthor ? byline : m.version}
        sx={{ fontSize: META_FONT_PX, color: 'text.secondary' }}
      >
        {showAuthor ? (
          <>
            <Box component="span" sx={{ position: 'relative', zIndex: 1 }}>
              <AuthorLink authorField={m.author} mod={m} profile={profile} />
            </Box>
            {m.version ? ` · ${m.version}` : null}
          </>
        ) : (
          m.version
        )}
      </Typography>
      {statsLine ? (
        <Typography
          className="card-stats"
          noWrap={true}
          sx={{ fontSize: META_FONT_PX, color: 'text.secondary', '& .card-stat': CARD_STAT_SX }}
        >
          {cardStats.length > 0 ? <StatParts stats={cardStats} /> : '\u00a0'}
        </Typography>
      ) : null}
    </>
  )
}
