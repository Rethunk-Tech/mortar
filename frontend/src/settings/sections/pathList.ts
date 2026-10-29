// Joins the folders the way the settings store them: one path-list string, ';' on Windows and ':' elsewhere.
export const separator = navigator.userAgent.includes('Windows') ? ';' : ':'
export const split = (value: string) => (value === '' ? [] : value.split(separator))

const SEGMENT = /[\\/]/

/**
 * Each folder's shortest ending, at least two segments, that no other folder in the list shares, after "…/" when
 * it is shorter than the whole path. Folders under one long parent otherwise all read as that parent.
 */
export function distinctTails(folders: string[]): string[] {
  const parts = folders.map((dir) => dir.split(SEGMENT).filter(Boolean))
  return folders.map((dir, i) => {
    const own = parts[i] ?? []
    const sep = dir.includes('\\') && !dir.includes('/') ? '\\' : '/'
    for (let k = Math.min(2, own.length); k < own.length; k++) {
      const tail = own.slice(-k).join(sep)
      if (parts.every((other, j) => j === i || other.slice(-k).join(sep) !== tail)) {
        return `…${sep}${tail}`
      }
    }
    return dir
  })
}
