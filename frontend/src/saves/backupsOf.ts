import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'

// The backups, in the order given, that hold the save folder.
function backupsOf(all: readonly Backup[], folder: string): Backup[] {
  return all.filter((b) => (b.saves ?? []).some((s) => s.folder === folder))
}

export { backupsOf }
