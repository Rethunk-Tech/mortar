import { Box, List } from '@mui/material'
import { useEffect } from 'react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { acknowledgeUpdateCaution } from '../../launch/autoUpdate.ts'
import { entryOf, modId } from '../lookup.ts'
import { useUpdates } from '../updates.ts'
import { Row } from './Row.tsx'
import { installedCaution } from './wants.ts'

export function ReviewList({
  list,
  profile,
  acked,
  onAck,
}: {
  list: Update[]
  profile: Profile
  acked: Record<string, boolean>
  onAck: (id: string, on: boolean) => void
}) {
  const focus = useUpdates((s) => s.focusKey)
  // Opened from one mod's update badge: its row comes into view once, then the list scrolls freely.
  useEffect(() => {
    if (focus === '') {
      return
    }
    document
      .querySelector(`[data-update-key="${CSS.escape(focus)}"]`)
      ?.scrollIntoView({ block: 'center' })
    useUpdates.setState({ focusKey: '' })
  }, [focus])
  return (
    <List disablePadding={true}>
      {list.map((u) => {
        const caution = installedCaution(u)
        const id = modId(u)
        const picture = entryOf(profile, u.key)?.source.picture
        return (
          <Box key={id} data-update-key={u.key}>
            <Row
              update={u}
              profileId={profile.id}
              caution={caution}
              acked={acked[id] === true}
              onAck={(on) => {
                onAck(id, on)
                acknowledgeUpdateCaution(profile.id, u, on)
              }}
              {...(picture === undefined ? {} : { picture })}
            />
          </Box>
        )
      })}
    </List>
  )
}
