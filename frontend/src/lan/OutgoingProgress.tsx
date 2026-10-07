import { msg } from '@lingui/core/macro'
import { useEffect, useRef } from 'react'
import type { OutgoingTransfer } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/models.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { i18n } from '../i18n/index.ts'
import { useToasts } from '../toasts/store.ts'
import { useIncomingShares } from './incoming.ts'

const KINDS: Record<string, 'info' | 'success' | 'error'> = { sending: 'info', done: 'success' }

function lineOf(transfer: OutgoingTransfer): string {
  const { peer, profile, current, total, bytes, totalBytes, state, reason } = transfer
  if (state === 'done') {
    return i18n._(msg`${peer} has all the files for ${profile}`)
  }
  if (state !== 'sending') {
    const why = state === 'cancelled' ? i18n._(msg`${peer} cancelled it`) : (reason ?? '')
    return i18n._(msg`Sending to ${peer} stopped: ${why}`)
  }
  const line = i18n._(msg`Sending to ${peer}: ${current} of ${total} files`)
  return totalBytes > 0 ? `${line} · ${formatBytes(bytes)} of ${formatBytes(totalBytes)}` : line
}

// A paired computer pulling a sent profile's files is one toast per transfer, updated in place, so the sender
// keeps using Mortar meanwhile. A finished transfer clears itself; a failed one stays until dismissed.
export function OutgoingProgress() {
  const outgoing = useIncomingShares((shares) => shares.outgoing)
  const toastOf = useRef(new Map<number, number>())
  const lastState = useRef(new Map<number, string>())

  useEffect(() => {
    for (const transfer of Object.values(outgoing)) {
      const { id, state } = transfer
      const title = lineOf(transfer)
      const kind = KINDS[state] ?? 'error'
      const toastId = toastOf.current.get(id)
      const finishedWell = state === 'done' && lastState.current.get(id) !== 'done'
      const sticky = finishedWell ? { sticky: false } : {}
      if (toastId === undefined) {
        toastOf.current.set(id, useToasts.getState().push({ kind, title, sticky: true }))
        if (finishedWell) {
          useToasts.getState().update(toastOf.current.get(id) ?? 0, sticky)
        }
      } else {
        useToasts.getState().update(toastId, { kind, title, ...sticky })
      }
      lastState.current.set(id, state)
    }
  }, [outgoing])
  return null
}
