// Reads a store key the way Go's store.NexusFile does: `nexus-<mod>-<file>`, digits only, trailing text allowed.
const LEADING_INT = /^(\d+)(.*)$/s

export function parseNexusKey(key: string): { modID: number; fileID: number } | null {
  if (!key.startsWith('nexus-')) {
    return null
  }
  const mod = LEADING_INT.exec(key.slice('nexus-'.length))
  const afterMod = mod?.[2] ?? ''
  if (!(mod && afterMod.startsWith('-'))) {
    return null
  }
  const file = LEADING_INT.exec(afterMod.slice(1))
  if (!file) {
    return null
  }
  return { modID: Number(mod[1]), fileID: Number(file[1]) }
}
