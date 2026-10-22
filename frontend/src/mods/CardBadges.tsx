import { Chip } from '@mui/material'
import { memo } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { CompatChip } from './CompatChip.tsx'
import { ExtraFilesChip } from './ExtraFilesChip.tsx'
import { LibrariesChip } from './LibrariesChip.tsx'
import type { ListRow } from './listColumns.ts'
import { sameLibraries } from './listRows.ts'
import { ModMenu } from './ModMenu.tsx'
import { OverlayCountChip } from './OverlayRow.tsx'
import {
  LastRunBadge,
  LiveBadge,
  NexusGoneBadge,
  PinBadge,
  ProblemBadge,
  UpdateBadge,
} from './parts.tsx'

const TAG_MAX_PX = 96

// The badges read their own stores, so a row's size or page details arriving re-renders the card's text, not them.
export const CardBadges = memo(
  function CardBadgesView({
    mod: m,
    profile,
    libraries,
    tag,
  }: {
    mod: Mod
    profile: Profile
    libraries: ListRow['libraries']
    tag: string
  }) {
    return (
      <>
        <LibrariesChip libraries={libraries} />
        <ExtraFilesChip mod={m} profile={profile} />
        <OverlayCountChip mod={m} profile={profile} />
        <PinBadge mod={m} />
        <CompatChip mod={m} />
        <UpdateBadge mod={m} />
        <NexusGoneBadge mod={m} />
        <ProblemBadge mod={m} />
        <LiveBadge mod={m} />
        <LastRunBadge mod={m} />
        {tag ? <Chip size="small" label={tag} title={tag} sx={{ maxWidth: TAG_MAX_PX }} /> : null}
        <ModMenu mod={m} />
      </>
    )
  },
  (a, b) =>
    a.mod === b.mod &&
    a.profile === b.profile &&
    a.tag === b.tag &&
    sameLibraries(a.libraries, b.libraries),
)
