import { useLingui } from '@lingui/react/macro'
import { Box, ToggleButton, ToggleButtonGroup, Typography } from '@mui/material'
import { userModCount } from '../profiles/count.ts'
import { profileById, useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { TipBanner } from '../tips/TipBanner.tsx'
import { ListFormat } from './ListFormat.tsx'
import { FilePane, LinkPane, NearbyPane, NexusPane, ThunderstorePane } from './MethodPanes.tsx'
import type { Destination } from './methods.ts'
import type { useShareBuild } from './useShareBuild.ts'

// The profile's own mod count, or the built list's while the profile is not loaded.
function useShareTotal({ profileId, info }: ReturnType<typeof useShareBuild>) {
  return useProfiles((s) => {
    const p = profileById(profileId)(s)
    return p ? userModCount(p) : (info?.count ?? 0)
  })
}

function MortarPanel({ built }: { built: ReturnType<typeof useShareBuild> }) {
  const { t } = useLingui()
  const { format, setFormat, info, include, setInclude } = built
  const total = useShareTotal(built)
  if (!info) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <ToggleButtonGroup
        exclusive={true}
        size="small"
        value={format}
        aria-label={t`How to send it`}
        onChange={(_, next: typeof format | null) => {
          if (next) {
            setFormat(next)
          }
        }}
        fullWidth={true}
      >
        <ToggleButton value="link" sx={{ px: space.pad }}>
          {t`Link`}
        </ToggleButton>
        <ToggleButton value="file" sx={{ px: space.pad }}>
          {t`File`}
        </ToggleButton>
      </ToggleButtonGroup>
      {format === 'link' ? (
        <TipBanner tip="share" inline={true}>
          {t`A share link names this profile and where each mod comes from, not the files themselves.`}
        </TipBanner>
      ) : null}
      {format === 'link' ? (
        <LinkPane
          info={info}
          total={total}
          include={include}
          onInclude={setInclude}
          onFile={() => setFormat('file')}
        />
      ) : (
        <FilePane
          info={info}
          total={total}
          include={include}
          onInclude={setInclude}
          saved={built.saved}
        />
      )}
    </Box>
  )
}

// The one result for the chosen destination.
export function DestinationPanel({
  built,
  destination,
}: {
  built: ReturnType<typeof useShareBuild>
  destination: Destination
}) {
  const { t } = useLingui()
  const { profileId, game, info, include, setInclude } = built
  const total = useShareTotal(built)
  if (!info) {
    return null
  }
  if (info.count === 0 && info.leftOut.length === 0) {
    return (
      <Typography role="status" sx={{ fontSize: 'medium.fontSize', color: 'warning.light' }}>
        {t`There are no mods to share.`}
      </Typography>
    )
  }
  switch (destination) {
    case 'mortar':
      return <MortarPanel built={built} />
    case 'nexus':
      return <NexusPane game={game} profileId={profileId} />
    case 'thunderstore':
      return <ThunderstorePane game={game} profileId={profileId} />
    case 'nearby':
      return (
        <NearbyPane
          game={game}
          profileId={profileId}
          info={info}
          total={total}
          include={include}
          onInclude={setInclude}
        />
      )
    default:
      return <ListFormat />
  }
}
