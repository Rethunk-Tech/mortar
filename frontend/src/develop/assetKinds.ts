import { useLingui } from '@lingui/react/macro'
import { readStored } from '@rethunk/react-load-hooks'
import { useEffect, useMemo, useState } from 'react'
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

export const CONTENT_KIND = 'content'
export const CONTENT_PARAM = 'mortarContent'

// What a pack of the kind needs to load anything: its loader, or for the content pack any one folder's mod.
const needs = (k: AssetKind) => (k.id === CONTENT_KIND ? (k.recommends ?? []) : [k.dependency])

// A lookup that fails (offline) counts the package as there; only one the source does not list is missing. A package
// that is neither on Thunderstore nor one of the player's projects makes a pack nothing loads.
async function missingPackages(game: string, packages: string[]): Promise<Set<string>> {
  const own = new Set(Object.values(useProjectInfo.getState().byDir).map((info) => info.package))
  const unique = [...new Set(packages)]
  const found = await Promise.all(
    unique.map((pkg) =>
      own.has(pkg)
        ? true
        : findPackage(game, '', pkg).then(
            (item) => item !== undefined,
            () => true,
          ),
    ),
  )
  return new Set(unique.filter((_, i) => !found[i]))
}

/** The form less the content folders whose mod is missing, and less the content question when none is left. */
function withOfferedContent(form: Form | null, missing: ReadonlySet<string>): Form | null {
  const gone = new Set(
    (form?.recommends ?? [])
      .filter((r) => r.param === CONTENT_PARAM && missing.has(r.package))
      .flatMap((r) => r.values ?? []),
  )
  if (!form || gone.size === 0) {
    return form
  }
  const parameters = (form.parameters ?? []).flatMap((p) => {
    if (p.id !== CONTENT_PARAM) {
      return [p]
    }
    const choices = (p.choices ?? []).filter((c) => !gone.has(c.value))
    return choices.length > 0 ? [{ ...p, choices }] : []
  })
  return { ...form, parameters }
}

const NONE_MISSING: ReadonlySet<string> = new Set()

/** The game's asset kinds, and the form for the kind the draft picked (the template's for a code mod; null while it
 * loads), less the content folders whose mod is missing. Picking a kind fills its questions as the template's are
 * filled: remembered answers first, what was typed kept. */
export function useAssetKinds(
  {
    open,
    game,
    kind,
    templateForm,
  }: { open: boolean; game: string; kind: string; templateForm: Form | null },
  setDraft: (fn: (d: Draft) => Draft) => void,
) {
  const { t } = useLingui()
  const [kinds, setKinds] = useState<AssetKind[]>([])
  const [missing, setMissing] = useState<ReadonlySet<string>>(NONE_MISSING)
  const [form, setForm] = useState<Form | null>(null)
  useEffect(() => {
    if (open) {
      AssetKinds(game)
        .then(async (found) => {
          const all = found ?? []
          const gone = await missingPackages(game, all.flatMap(needs))
          setMissing(gone)
          setKinds(all.filter((k) => needs(k).some((pkg) => !gone.has(pkg))))
        })
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
  const shown = useMemo(
    () => withOfferedContent(kind ? form : templateForm, missing),
    [kind, form, templateForm, missing],
  )
  return { kinds, form: shown }
}
