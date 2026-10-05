import type { PluginCopy } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { EnableRef } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

// Every component of every copy that loses the clash: a package holds several mods, and the clash stays while any
// one of them is enabled.
function losingRefs(
  copies: readonly PluginCopy[],
  keep: string,
  mods: readonly { key: string; id: string }[],
): EnableRef[] {
  return copies
    .filter((copy) => copy.key !== keep)
    .flatMap((copy) => {
      const ids = mods.filter((m) => m.key === copy.key).map((m) => m.id)
      return (ids.length > 0 ? ids : [copy.id]).map((id) => ({ key: copy.key, id }))
    })
}

export { losingRefs }
