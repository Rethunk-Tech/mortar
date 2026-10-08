import { i18n } from '../i18n/index.ts'

const authorPartSplit = /\s*(?:[,&]|\band\b)\s*/i

function normalizeAuthorName(name: string): string {
  return name.trim().replace(/\s+/g, ' ').toLowerCase()
}

function splitManifestAuthors(field: string): string[] {
  const trimmed = field.trim()
  if (trimmed === '') {
    return []
  }
  const seen = new Set<string>()
  return trimmed
    .split(authorPartSplit)
    .map((part) => part.trim().replace(/\s+/g, ' '))
    .filter((part) => {
      const key = normalizeAuthorName(part)
      if (key === '' || seen.has(key)) {
        return false
      }
      seen.add(key)
      return true
    })
}

/** The field's names as a list for the active locale; `element` parts are names, `literal` parts the joiners. */
function authorListParts(field: string): { type: 'element' | 'literal'; value: string }[] {
  return new Intl.ListFormat(i18n.locale, { type: 'conjunction' }).formatToParts(
    splitManifestAuthors(field),
  )
}

/** Author field as one display string: each name once, joined for the active locale. */
function formatAuthors(field: string): string {
  return authorListParts(field)
    .map((part) => part.value)
    .join('')
}

function authorFieldIncludes(field: string, author: string): boolean {
  const want = normalizeAuthorName(author)
  if (want === '') {
    return false
  }
  return splitManifestAuthors(field).some((part) => normalizeAuthorName(part) === want)
}

export {
  authorFieldIncludes,
  authorListParts,
  formatAuthors,
  normalizeAuthorName,
  splitManifestAuthors,
}
