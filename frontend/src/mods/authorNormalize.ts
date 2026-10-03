const authorPartSplit = /\s*[,&]\s*/

function normalizeAuthorName(name: string): string {
  return name.trim().replace(/\s+/g, ' ').toLowerCase()
}

function splitManifestAuthors(field: string): string[] {
  const trimmed = field.trim()
  if (trimmed === '') {
    return []
  }
  return trimmed
    .split(authorPartSplit)
    .map((part) => part.trim().replace(/\s+/g, ' '))
    .filter((part) => part !== '')
}

function authorFieldIncludes(field: string, author: string): boolean {
  const want = normalizeAuthorName(author)
  if (want === '') {
    return false
  }
  return splitManifestAuthors(field).some((part) => normalizeAuthorName(part) === want)
}

export { authorFieldIncludes, normalizeAuthorName, splitManifestAuthors }
