import { useLingui } from '@lingui/react/macro'
import { useEffect, useState } from 'react'
import type { AssetKind } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'
import type { Form } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  AssetForm,
  AssetKinds,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { readStored } from '../shell/useStoredState.ts'
import { reportError } from '../toasts/report.ts'
import { type Draft, firstDraft, isRemembered, REMEMBERED } from './draft.ts'

/** The game's asset kinds, and the form for the one the draft picked (null for a code mod or while it loads). Picking a
 * kind fills its questions as the template's are filled: remembered answers first, what was typed kept. */
export function useAssetKinds(
  open: boolean,
  game: string,
  kind: string,
  setDraft: (fn: (d: Draft) => Draft) => void,
) {
  const { t } = useLingui()
  const [kinds, setKinds] = useState<AssetKind[]>([])
  const [form, setForm] = useState<Form | null>(null)
  useEffect(() => {
    if (open) {
      AssetKinds(game)
        .then((found) => setKinds(found ?? []))
        .catch(() => setKinds([]))
    }
  }, [open, game])
  useEffect(() => {
    setForm(null)
    if (!kind) {
      return
    }
    let stale = false
    AssetForm(game, kind)
      .then((read) => {
        if (stale) {
          return
        }
        setForm(read)
        setDraft((d) => {
          const fresh = firstDraft(read, readStored(REMEMBERED, {}, isRemembered))
          const typed = Object.fromEntries(Object.entries(d.values).filter(([, v]) => v !== ''))
          return { ...d, values: { ...fresh.values, ...typed } }
        })
      })
      .catch(reportError(t`Could not read that kind of mod`))
    return () => {
      stale = true
    }
  }, [game, kind, setDraft, t])
  return { kinds, form }
}
