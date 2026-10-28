/** The last part of a path with either separator; a trailing separator is ignored. */
export const baseName = (path: string) =>
  path
    .replace(/[\\/]+$/, '')
    .split(/[\\/]/)
    .pop() ?? ''
