import { useLingui } from '@lingui/react/macro'
import { readStored } from '@rethunk/react-load-hooks'
import { useEffect, useMemo, useState } from 'react'
import type { Form } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { AssetForm } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { reportError } from '../toasts/report.ts'
import { useProjectInfo } from './builds.ts'
import { findPackage } from './dependencies.ts'
import { CATEGORIES_PARAM, type Draft, firstDraft, isRemembered, REMEMBERED } from './draft.ts'

export const CONTENT_KIND = 'content'
export const CONTENT_PARAM = 'mortarContent'

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
      .filter((r) => r.param === CONTENT_PARAM && !r.optional && missing.has(r.package))
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

/** The no-code form with the template's categories before its content question, for the pack to be listed under. */
function withCategories(form: Form | null, templateForm: Form | null): Form | null {
  const categories = templateForm?.parameters?.find((p) => p.id === CATEGORIES_PARAM)
  if (!(form && categories)) {
    return form
  }
  const parameters = form.parameters ?? []
  const at = parameters.findIndex((p) => p.id === CONTENT_PARAM)
  return {
    ...form,
    parameters: parameters.toSpliced(at < 0 ? parameters.length : at, 0, categories),
  }
}

const NONE_MISSING: ReadonlySet<string> = new Set()

/** The form for the draft's kind (the template's for a code mod; null while the no-code one loads), less the content
 * folders whose mod is missing, and whether any content is on offer to make a mod without code. Picking the no-code
 * kind fills its questions as the template's are filled: remembered answers first, what was typed kept. */
export function useWizardForm(
  { game, kind, templateForm }: { game: string; kind: string; templateForm: Form | null },
  setDraft: (fn: (d: Draft) => Draft) => void,
) {
  const { t } = useLingui()
  const [missing, setMissing] = useState<ReadonlySet<string>>(NONE_MISSING)
  const [form, setForm] = useState<Form | null>(null)
  useEffect(() => {
    const needed = (templateForm?.recommends ?? [])
      .filter((r) => r.param === CONTENT_PARAM && !r.optional)
      .map((r) => r.package)
    let stale = false
    missingPackages(game, needed)
      .then((gone) => {
        if (!stale) {
          setMissing(gone)
        }
      })
      .catch(() => undefined)
    return () => {
      stale = true
    }
  }, [game, templateForm])
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
    () => withOfferedContent(kind ? withCategories(form, templateForm) : templateForm, missing),
    [kind, form, templateForm, missing],
  )
  const offered = useMemo(
    () =>
      withOfferedContent(templateForm, missing)?.parameters?.some((p) => p.id === CONTENT_PARAM),
    [templateForm, missing],
  )
  return { form: shown, noCodeOffered: offered === true }
}
