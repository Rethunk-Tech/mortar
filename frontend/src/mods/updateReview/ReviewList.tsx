import { Box } from '@mui/material'
import type { Update } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type {
  Mod,
  Profile,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { acknowledgeUpdateCaution } from '../../launch/autoUpdate.ts'
import { download } from '../../queue/actions.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { entryOf, modId } from '../lookup.ts'
import { Row } from './Row.tsx'
import { installedCaution, updateWant } from './wants.ts'

export function ReviewList({
  list,
  profile,
  mods,
  acked,
  include,
  onAck,
  onInclude,
  onPropagate,
}: {
  list: Update[]
  profile: Profile
  mods: Mod[]
  acked: Record<string, boolean>
  include: Record<string, boolean>
  onAck: (id: string, on: boolean) => void
  onInclude: (id: string, on: boolean) => void
  onPropagate: (update: Update) => void
}) {
  return (
    <Box role="list">
      {list.map((u) => {
        const caution = installedCaution(mods, u)
        const id = modId(u)
        const picture = entryOf(profile, u.key)?.source.picture
        return (
          <Row
            key={id}
            update={u}
            profileId={profile.id}
            caution={caution}
            acked={acked[id] === true}
            included={include[id] !== false}
            onAck={(on) => {
              onAck(id, on)
              acknowledgeUpdateCaution(profile.id, u, on)
            }}
            onInclude={(on) => onInclude(id, on)}
            {...(picture === undefined ? {} : { picture })}
            onUpdateAll={() => {
              download([updateWant(u)])
                .then((added) => {
                  if (added) {
                    onPropagate(u)
                  }
                })
                .catch(reportUnexpected)
            }}
          />
        )
      })}
    </Box>
  )
}
