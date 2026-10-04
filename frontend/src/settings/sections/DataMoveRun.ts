import { msg } from '@lingui/core/macro'
import type { MoveEstimate } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import {
  MoveDataFolder,
  UsageProgress,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import { i18n } from '../../i18n/index.ts'
import { type InlineError, inlineError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'

const MOVE_PROGRESS_INTERVAL = 200

export interface MoveState {
  dest: string
  estimate: MoveEstimate
}

export function moveDataFolder(options: {
  move: MoveState
  setMove: (value: MoveState | null) => void
  setMoveError: (value: InlineError | null) => void
  setMoving: (value: boolean) => void
  setMoveProgress: (value: {
    files: number
    totalFiles: number
    bytes: number
    totalBytes: number
  }) => void
}) {
  const { move, setMove, setMoveError, setMoving, setMoveProgress } = options
  setMoving(true)
  setMoveError(null)
  const poll = globalThis.setInterval(() => {
    UsageProgress().then(
      (progress) => setMoveProgress(progress),
      () => undefined,
    )
  }, MOVE_PROGRESS_INTERVAL)
  MoveDataFolder(move.dest)
    .then(() => {
      setMove(null)
      useToasts.getState().push({ kind: 'success', title: i18n._(msg`Data folder moved`) })
    })
    .catch((err: unknown) => setMoveError(inlineError(err)))
    .finally(() => {
      globalThis.clearInterval(poll)
      setMoving(false)
    })
}
