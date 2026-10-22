import { Box, Typography } from '@mui/material'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { parseCardStats, useModCardStats } from '../browse/stats.ts'
import { useSettings } from '../settings/store.ts'
import { AuthorLink } from './AuthorLink.tsx'
import { formatAuthors } from './authorNormalize.ts'
import type { ListRow } from './listColumns.ts'

const META_FONT_PX = 12

// A card's second line (author and version) and, when Card details are ticked, the line of chosen details under it.
export function CardMeta({
  mod: m,
  row,
  profile,
  statsLine,
}: {
  mod: Mod
  row: ListRow
  profile: Profile
  statsLine: boolean
}) {
  const showAuthor = useSettings((s) => s.showAuthorOnCards) !== false
  const stats = useModCardStats(row, parseCardStats(useSettings((s) => s.browseCardStats))).join(
    ' · ',
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
          title={stats}
          sx={{ fontSize: META_FONT_PX, color: 'text.secondary' }}
        >
          {stats || '\u00a0'}
        </Typography>
      ) : null}
    </>
  )
}
