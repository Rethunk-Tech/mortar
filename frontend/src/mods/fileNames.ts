import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

// A per-file entry's key is its download's key, '#', then the file's path (profile.fileKeySep); its name is that path.
const SEP = '#'

const dirOf = (file: string) => file.slice(0, file.lastIndexOf('/') + 1)

function sharedDir(files: readonly string[]): string {
  let dir = dirOf(files[0] ?? '')
  for (const f of files) {
    while (!f.startsWith(dir)) {
      dir = dirOf(dir.slice(0, -1))
    }
  }
  return dir
}

// Per-file entries named below the folder every file of their download shares, so
// bin/x64/plugins/cyber_engine_tweaks/mods/foo/init.lua reads init.lua. A name the player changed is kept.
export function shortFileNames(mods: Mod[]): Mod[] {
  const files = new Map<string, string[]>()
  for (const m of mods) {
    const at = m.key.indexOf(SEP)
    if (at > 0) {
      const item = m.key.slice(0, at)
      files.set(item, [...(files.get(item) ?? []), m.key.slice(at + 1)])
    }
  }
  if (files.size === 0) {
    return mods
  }
  return mods.map((m) => {
    const at = m.key.indexOf(SEP)
    const file = at > 0 ? m.key.slice(at + 1) : ''
    const dir = file === '' ? '' : sharedDir(files.get(m.key.slice(0, at)) ?? [])
    return dir !== '' && m.name === file ? { ...m, name: file.slice(dir.length) } : m
  })
}
