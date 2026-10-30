import { useLingui } from '@lingui/react/macro'
import { readStored } from '@rethunk/react-load-hooks'
import { useEffect, useState } from 'react'
import type { AssetKind } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'
import type { Form } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  AssetForm,
  AssetKinds,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { reportError } from '../toasts/report.ts'
import { useProjectInfo } from './builds.ts'
import { findPackage } from './dependencies.ts'
import { type Draft, firstDraft, isRemembered, REMEMBERED } from './draft.ts'

// A lookup that fails (offline) leaves the kind offered; only a loader the source does not list hides it.
// A kind whose loader is neither on Thunderstore nor one of the player's projects makes a pack nothing loads.
async function offered(game: string, kinds: AssetKind[]): Promise<AssetKind[]> {
  const own = new Set(Object.values(useProjectInfo.getState().byDir).map((info) => info.package))
  const found = await Promise.all(
    kinds.map((k) =>
      own.has(k.dependency)
        ? true
        : findPackage(game, '', k.dependency).then(
            (item) => item !== undefined,
            () => true,
          ),
    ),
  )
  return kinds.filter((_, i) => found[i])
}

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
        .then((found) => offered(game, found ?? []))
        .then(setKinds)
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
