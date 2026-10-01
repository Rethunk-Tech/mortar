export interface NexusPageMark {
  kind: 'removed' | 'hidden' | ''
  date: string
}

export function nexusPageMark(
  status: string | undefined,
  available: boolean | undefined,
  updated?: string,
  created?: string,
): NexusPageMark {
  const day = (s?: string) => {
    const v = (s ?? '').trim()
    if (v === '') {
      return ''
    }
    const t = v.indexOf('T')
    return t > 0 ? v.slice(0, t) : v.slice(0, 10)
  }
  const date = day(updated) || day(created)
  if (available === false) {
    return { kind: 'hidden', date }
  }
  const st = (status ?? '').trim().toLowerCase()
  if (st !== '' && st !== 'published') {
    return { kind: 'removed', date }
  }
  return { kind: '', date: '' }
}

export function offersNexusDownload(status?: string, available?: boolean): boolean {
  return nexusPageMark(status, available).kind === ''
}

export function goneCaption(
  mark: NexusPageMark,
  labels: { hidden: string; hiddenDated: string; removed: string; removedDated: string },
): string {
  if (mark.kind === 'hidden') {
    return mark.date ? labels.hiddenDated : labels.hidden
  }
  if (mark.kind === 'removed') {
    return mark.date ? labels.removedDated : labels.removed
  }
  return ''
}
