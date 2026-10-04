import { useEffect, useRef, useState } from 'react'
import {
  GmcmMenu,
  GmcmResult,
  PendingGmcm,
  SetGmcmEdits,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../toasts/report.ts'
import {
  draftMap,
  type GmcmCapture,
  pendingEdits,
  type GmcmResult as Result,
} from './configMenu.ts'
import { openTarget } from './storeView.ts'

export function useGmcmMenu(uniqueId: string, open: boolean) {
  const [capture, setCapture] = useState<GmcmCapture | null>(null)
  const [drafts, setDrafts] = useState<Record<string, unknown>>({})
  const [result, setResult] = useState<Result | null>(null)
  const [pageId, setPageId] = useState('')
  const gen = useRef(0)
  useEffect(() => {
    gen.current += 1
    if (!open) {
      setCapture(null)
      setDrafts({})
      setResult(null)
      setPageId('')
      return
    }
    const seq = gen.current
    const t = openTarget()
    if (!t) {
      return
    }
    Promise.all([
      GmcmMenu(t.game, t.id, uniqueId).catch(() => null),
      PendingGmcm(t.game, t.id, uniqueId).catch(() => ({ schema: 1, edits: [] })),
      GmcmResult(t.game, t.id, uniqueId).catch(() => ({ applied: 0, skipped: [] })),
    ])
      .then(([menu, pending, last]) => {
        if (seq !== gen.current) {
          return
        }
        const cap = menu as GmcmCapture | null
        setCapture(cap)
        setDrafts(draftMap(pending))
        setResult(last as Result)
        setPageId(cap?.pages?.[0]?.id ?? '')
      })
      .catch(() => undefined)
  }, [open, uniqueId])
  const persist = (next: Record<string, unknown>, cap: GmcmCapture | null) => {
    const t = openTarget()
    if (!(t && cap)) {
      return
    }
    SetGmcmEdits(t.game, t.id, uniqueId, pendingEdits(cap, next)).catch(reportUnexpected)
  }
  const change = (key: string, value: unknown) => {
    setDrafts((cur) => {
      const next = { ...cur, [key]: value }
      persist(next, capture)
      return next
    })
  }
  const save = () => persist(drafts, capture)
  const discard = () => {
    const t = openTarget()
    if (!t) {
      return
    }
    setDrafts({})
    SetGmcmEdits(t.game, t.id, uniqueId, []).catch(reportUnexpected)
  }
  return { capture, drafts, result, pageId, setPageId, change, save, discard }
}
