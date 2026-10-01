import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { Skip } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { accent } from '../mods/paper.ts'
import { LetterTile } from '../mods/parts.tsx'
import { useProfiles } from '../profiles/store.ts'
import { profileOf } from '../queue/totals.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useInstall } from './store.ts'

const tile = (i: Item) => ({
  uniqueId: i.repo || String(i.modId),
  name: i.name || i.repo || String(i.modId),
  picture: '',
})

function Title({ item, size }: { item: Item; size: number }) {
  const { t } = useLingui()
  const profile = useProfiles((s) => profileOf(item, s.game?.id, s.profiles))
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
      <Typography noWrap={true} sx={{ fontSize: size, fontWeight: 600 }}>
        {item.name || item.fileName}
      </Typography>
      {profile === null ? null : (
        <Typography noWrap={true} sx={{ fontSize: 12, color: 'text.secondary' }}>
          {profile ? t`into ${profile}` : t`into a deleted profile`}
        </Typography>
      )}
    </Box>
  )
}

export function NeedsRootCallout({ item }: { item: Item }) {
  const { t } = useLingui()
  const openRemap = useInstall((s) => s.openRemap)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === item.profileId))
  const ask = item.remap
  if (!ask) {
    return null
  }
  const profileName = profile?.name ?? ''
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        p: '14px',
        bgcolor: accent.fill,
        border: '1px solid',
        borderColor: 'primary.main',
        borderRadius: '8px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <LetterTile mod={tile(item)} size={44} />
        <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          <Typography
            sx={{
              fontSize: 12,
              fontWeight: 700,
              letterSpacing: '0.06em',
              textTransform: 'uppercase',
              color: 'primary.main',
            }}
          >
            {t`Choose a folder`}
          </Typography>
          <Title item={item} size={16} />
        </Box>
        <Box sx={{ flexShrink: 0 }}>
          <Button
            variant="outlined"
            color="inherit"
            onClick={() => Skip(item.id).catch(reportUnexpected)}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Skip`}
          </Button>
        </Box>
      </Box>
      <Typography sx={{ fontSize: 13, lineHeight: 1.45 }}>
        {t`This archive has no SMAPI mod where Mortar expects one. Pick the folder that holds manifest.json.`}
      </Typography>
      <Button
        variant="contained"
        onClick={() =>
          openRemap({
            game: item.game,
            profileId: item.profileId,
            profileName,
            key: ask.key,
            source: ask.source,
            ask,
            queueId: item.id,
          })
        }
        sx={{ alignSelf: 'flex-start', whiteSpace: 'nowrap' }}
      >
        {t`Choose folder…`}
      </Button>
    </Box>
  )
}
