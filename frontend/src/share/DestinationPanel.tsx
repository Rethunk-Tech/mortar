import { useLingui } from '@lingui/react/macro'
import { Box, Button, ToggleButton, ToggleButtonGroup, Typography } from '@mui/material'
import { useState } from 'react'
import {
  FilePane,
  LinkPane,
  ListPane,
  NearbyPane,
  NexusPane,
  ThunderstorePane,
} from './MethodPanes.tsx'
import type { Destination } from './methods.ts'
import { PreviewDialog } from './PreviewDialog.tsx'
import type { useShareBuild } from './useShareBuild.ts'

function MortarPanel({ built }: { built: ReturnType<typeof useShareBuild> }) {
  const { t } = useLingui()
  const { profileId, keys, game, format, setFormat, info, include, setInclude } = built
  const [previewing, setPreviewing] = useState(false)
  if (!info) {
    return null
  }
  return (
    <>
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
        sx={{ alignSelf: 'flex-start', mb: 2.25 }}
      >
        <ToggleButton value="link" sx={{ px: 2.5 }}>
          {t`Link`}
        </ToggleButton>
        <ToggleButton value="file" sx={{ px: 2.5 }}>
          {t`File`}
        </ToggleButton>
      </ToggleButtonGroup>
      {format === 'link' ? (
        <LinkPane
          info={info}
          include={include}
          onInclude={setInclude}
          onFile={() => setFormat('file')}
        />
      ) : (
        <FilePane
          info={info}
          game={game}
          profileId={profileId}
          keys={keys}
          include={include}
          onInclude={setInclude}
        />
      )}
      <Box sx={{ flex: 1 }} />
      <Box sx={{ pt: 2 }}>
        <Button
          variant="text"
          disabled={info.tooLarge}
          onClick={() => setPreviewing(true)}
          sx={{ p: 0, minWidth: 0 }}
        >
          {t`Preview what they see`}
        </Button>
      </Box>
      <PreviewDialog info={info} open={previewing} onClose={() => setPreviewing(false)} />
    </>
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
  const { profileId, game, info } = built
  if (!info) {
    return null
  }
  if (info.count === 0) {
    return (
      <Typography role="status" sx={{ fontSize: 14, color: 'warning.light' }}>
        {t`No mods to share.`}
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
      return <NearbyPane game={game} profileId={profileId} />
    default:
      return <ListPane info={info} />
  }
}
