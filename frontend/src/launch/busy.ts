import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'

// The game process is starting or running, for one game or (without a game) any.
export const gameBusy = (status: Status | null, game?: string) =>
  (status?.state === State.Launching || status?.state === State.Running) &&
  (game === undefined || status.game === game)
