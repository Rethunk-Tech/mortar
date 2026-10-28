// The shape of frontend/src/settings/generated/credits.json, shared by its generator and its test.

export interface CreditEntry {
  name: string
  licence: string
  url: string
}

export function isCreditList(v: unknown): v is CreditEntry[] {
  if (!Array.isArray(v) || v.length === 0) {
    return false
  }
  return v.every((item) => {
    if (item === null || typeof item !== 'object' || Array.isArray(item)) {
      return false
    }
    const rec = item as Record<string, unknown>
    return (
      typeof rec.name === 'string' &&
      rec.name.length > 0 &&
      typeof rec.licence === 'string' &&
      rec.licence.length > 0 &&
      typeof rec.url === 'string' &&
      /^https?:\/\//.test(rec.url)
    )
  })
}
