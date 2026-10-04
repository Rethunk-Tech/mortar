import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { NewFoldersCallout } from './NewFoldersCallout.tsx'
import { OldFilesCallouts } from './OldFilesCallouts.tsx'

// What the library asks about above the mod list, beside the problem and update bars.
export function LibraryCallouts({ profile }: { profile: Profile }) {
  return (
    <>
      <OldFilesCallouts profile={profile} />
      <NewFoldersCallout profile={profile} />
    </>
  )
}
