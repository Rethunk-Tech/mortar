import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import {
  FilePane,
  LinkPane,
  ListPane,
  NearbyPane,
  NexusPane,
  ThunderstorePane,
} from './MethodPanes.tsx'
import type { useShareBuild } from './useShareBuild.ts'

// The one result for the selected method.
export function MethodPanel({ built }: { built: ReturnType<typeof useShareBuild> }) {
  const { t } = useLingui()
  const { profileId, keys, game, method, setMethod, info, include, setInclude } = built
  if (!info) {
    return null
  }
  return (
    <>
      {info.count === 0 ? (
        <Typography role="status" sx={{ fontSize: 14, color: 'warning.light' }}>
          {t`No mods to share.`}
        </Typography>
      ) : null}
      {info.count > 0 && method === 'link' ? (
        <LinkPane
          info={info}
          include={include}
          onInclude={setInclude}
          onFile={() => setMethod('file')}
        />
      ) : null}
      {info.count > 0 && method === 'file' ? (
        <FilePane
          info={info}
          game={game}
          profileId={profileId}
          keys={keys}
          include={include}
          onInclude={setInclude}
        />
      ) : null}
      {info.count > 0 && method === 'list' ? <ListPane info={info} /> : null}
      {info.count > 0 && method === 'nexus' ? (
        <NexusPane game={game} profileId={profileId} />
      ) : null}
      {info.count > 0 && method === 'nearby' ? (
        <NearbyPane game={game} profileId={profileId} />
      ) : null}
      {info.count > 0 && method === 'thunderstore' ? (
        <ThunderstorePane game={game} profileId={profileId} />
      ) : null}
    </>
  )
}
