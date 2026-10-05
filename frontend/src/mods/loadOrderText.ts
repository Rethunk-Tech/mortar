import type { Row } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loadorder/models.ts'

export function formatLoadOrderCopy(rows: readonly Row[]): string {
  return rows
    .map((row) => {
      const name = row.name.trim() === '' ? row.id : row.name
      return `${row.position}. ${name}`
    })
    .join('\n')
}

export function loadOrderEmptyKind(failed: boolean, count: number): 'error' | 'empty' | 'list' {
  if (failed) {
    return 'error'
  }
  return count === 0 ? 'empty' : 'list'
}
