import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'

// The game holds the open profile's mods; the backend refuses every change to them.
export const isLocked = (status: Status | null, openId: string) =>
  status?.state === State.Running && status.profile === openId
