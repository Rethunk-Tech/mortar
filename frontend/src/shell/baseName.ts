const TRAILING_SEPARATORS = /[\\/]+$/
const SEPARATOR = /[\\/]/

/** The last part of a path with either separator; a trailing separator is ignored. */
export const baseName = (path: string) =>
  path.replace(TRAILING_SEPARATORS, '').split(SEPARATOR).pop() ?? ''
