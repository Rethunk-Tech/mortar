interface ProblemReportSection {
  title: string
  count: number
  whyKeys?: readonly string[]
  lines?: readonly string[]
}

interface WhyEvidence {
  packId?: string
  target?: string
}

function whyKeysOf(evidence: readonly WhyEvidence[] | null | undefined): string[] {
  const keys: string[] = []
  for (const item of evidence ?? []) {
    const key = (item.target ?? '').trim() || (item.packId ?? '').trim()
    if (key !== '') {
      keys.push(key)
    }
  }
  return keys
}

function formatProblemReport(
  sections: readonly ProblemReportSection[],
  harmlessLabel: string,
  harmlessCount: number,
): string {
  const lines = sections.flatMap((section) => {
    const head = `${section.title}: ${section.count}`
    const keys = (section.whyKeys ?? []).filter((key) => key.trim() !== '')
    const extra = (section.lines ?? []).filter((line) => line.trim() !== '')
    const rest = [...keys.map((key) => `  ${key}`), ...extra.map((line) => `  ${line}`)]
    return rest.length === 0 ? [head] : [head, ...rest]
  })
  lines.push(`${harmlessLabel}: ${harmlessCount}`)
  return lines.join('\n')
}

export { formatProblemReport, whyKeysOf }
