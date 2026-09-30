import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'

// The game holds that profile's mods from Play on; vanilla launch has no profile and does not lock mods/.
export const isLocked = (status: Status | null, openId: string, startingProfile = '') =>
  (startingProfile !== '' && startingProfile === openId) ||
  ((status?.state === State.Launching || status?.state === State.Running) &&
    status.profile === openId)
