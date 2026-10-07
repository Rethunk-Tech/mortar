import { useLingui } from '@lingui/react/macro'
import { Chip, Tooltip } from '@mui/material'
import type { UpdateNote } from './smapiUpdateNotes.ts'

/** Mortar's answer to one line of SMAPI's "You can update" list, shown at the end of that line. */
export function UpdateNoteTag({ note }: { note: UpdateNote }) {
  const { t } = useLingui()
  const have = note.have ?? ''
  const [label, why] = {
    current: [
      t`Already installed`,
      t`Your download is already ${have}; the mod's manifest still carries the old number, so SMAPI keeps asking.`,
    ],
    skipped: [
      t`You skipped this version`,
      t`This profile skips this version, so Mortar does not offer it.`,
    ],
    pinned: [
      t`Pinned`,
      t`This mod is pinned in this profile, so Mortar does not offer updates for it.`,
    ],
    ignored: [t`Updates ignored`, t`This profile ignores updates for this mod.`],
    source: [t`Source skipped`, t`This profile skips updates from this source for this mod.`],
    prerelease: [t`Prerelease`, t`Mortar offers prereleases only when Settings allows them.`],
    unofficial: [t`Unofficial build`, t`Unofficial SMAPI-compatible builds are off in Settings.`],
    listed: [t`In Mortar's updates`, t`Mortar offers this update in the profile's update list.`],
    elsewhere: [
      t`Not downloadable here`,
      t`This version is only on a site Mortar cannot download from yet; open the link to get it.`,
    ],
  }[note.kind]
  return (
    <Tooltip title={why}>
      <Chip
        size="small"
        variant="outlined"
        tabIndex={0}
        label={t`Mortar: ${label}`}
        sx={{ ml: 1, flexShrink: 0, fontSize: 11, color: 'text.secondary' }}
      />
    </Tooltip>
  )
}
