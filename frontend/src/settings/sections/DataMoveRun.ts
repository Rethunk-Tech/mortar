import type { MoveEstimate } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import {
  MoveDataFolder,
  UsageProgress,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import { errorText } from '../../toasts/report.ts'

const MOVE_PROGRESS_INTERVAL = 200

export interface MoveState {
  dest: string
  estimate: MoveEstimate
}

export function moveDataFolder(options: {
  move: MoveState
  setMove: (value: MoveState | null) => void
  setMoveError: (value: string) => void
  setMoving: (value: boolean) => void
  setMoveProgress: (value: {
    files: number
    totalFiles: number
    bytes: number
    totalBytes: number
  }) => void
  moveErrorText: string
}) {
  const { move, setMove, setMoveError, setMoving, setMoveProgress, moveErrorText } = options
  setMoving(true)
  setMoveError('')
  const poll = globalThis.setInterval(() => {
    UsageProgress().then(
      (progress) => setMoveProgress(progress),
      () => undefined,
    )
  }, MOVE_PROGRESS_INTERVAL)
  MoveDataFolder(move.dest)
    .then(() => setMove(null))
    .catch((err: unknown) => setMoveError(errorText(err) || moveErrorText))
    .finally(() => {
      globalThis.clearInterval(poll)
      setMoving(false)
    })
}
