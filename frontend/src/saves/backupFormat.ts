const KiB = 1024
const MiB = KiB * KiB

export function formatBytes(n: number): string {
  if (n < KiB) {
    return `${n} B`
  }
  if (n < MiB) {
    return `${(n / KiB).toFixed(1)} KB`
  }
  return `${(n / MiB).toFixed(1)} MB`
}
