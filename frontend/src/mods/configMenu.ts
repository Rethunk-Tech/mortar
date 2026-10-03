export interface GmcmChoice {
  value: string
  label: string
}

export interface GmcmOption {
  index: number
  kind: string
  fieldId: string | null
  name: string
  tooltip: string
  value: unknown
  min: number | null
  max: number | null
  interval: number | null
  choices: GmcmChoice[] | null
  formatSamples: string[] | null
  editable: boolean
  titleScreenOnly: boolean
}

export interface GmcmPage {
  id: string
  title: string
  options: GmcmOption[]
}

export interface GmcmCapture {
  schema: number
  mod: { id: string; name: string; version: string }
  gmcmVersion: string
  capturedAt: string
  titleScreenOnlyDefault: boolean
  pages: GmcmPage[]
}

export interface GmcmEdit {
  page: string
  index: number
  kind: string
  fieldId: string
  name: string
  value: unknown
}

export interface GmcmPending {
  schema: number
  edits: GmcmEdit[]
}

export interface GmcmSkipped {
  edit: GmcmEdit
  reason: string
}

export interface GmcmResult {
  applied: number
  skipped: GmcmSkipped[]
}

export function optionKey(page: string, index: number): string {
  return `${page}/${index}`
}

function isAbsent(v: unknown): boolean {
  return v === null || v === undefined
}

export function valuesEqual(a: unknown, b: unknown): boolean {
  if (a === b) {
    return true
  }
  if (isAbsent(a) || isAbsent(b)) {
    return isAbsent(a) && isAbsent(b)
  }
  if (typeof a === 'number' && typeof b === 'number') {
    return a === b
  }
  return JSON.stringify(a) === JSON.stringify(b)
}

export function pendingEdits(capture: GmcmCapture, drafts: Record<string, unknown>): GmcmEdit[] {
  const out: GmcmEdit[] = []
  for (const page of capture.pages ?? []) {
    for (const opt of page.options ?? []) {
      const key = optionKey(page.id, opt.index)
      if (key in drafts && !valuesEqual(drafts[key], opt.value)) {
        out.push({
          page: page.id,
          index: opt.index,
          kind: opt.kind,
          fieldId: opt.fieldId ?? '',
          name: opt.name,
          value: drafts[key],
        })
      }
    }
  }
  return out
}

export function draftMap(pending: GmcmPending | null): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const edit of pending?.edits ?? []) {
    out[optionKey(edit.page, edit.index)] = edit.value
  }
  return out
}

export function optionDraft(
  page: string,
  opt: GmcmOption,
  drafts: Record<string, unknown>,
): unknown {
  const key = optionKey(page, opt.index)
  return key in drafts ? drafts[key] : opt.value
}
