import type { SDKNeed } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'

// major, minor, feature band (the hundreds of the patch) and patch, as .NET's roll-forward compares them.
const VERSION = /^(\d+)\.(\d+)\.(\d+)/

function parts(v: string): [number, number, number, number] | null {
  const m = VERSION.exec(v)
  if (!m) {
    return null
  }
  const patch = Number(m[3])
  return [Number(m[1]), Number(m[2]), Math.floor(patch / 100), patch]
}

const PATCH_FIXED = 3

// How many leading parts must match exactly under each rollForward; the rest only need to be at least the pin's.
const FIXED: Record<string, number> = {
  patch: 3,
  latestPatch: 3,
  feature: 2,
  latestFeature: 2,
  minor: 1,
  latestMinor: 1,
  major: 0,
  latestMajor: 0,
  disable: 4,
}

/** Whether one of the installed SDKs meets a template's global.json pin; a template that pins none needs any. */
export function sdkSatisfies(need: SDKNeed, installed: string[]): boolean {
  const want = parts(need.version)
  if (!want) {
    return installed.length > 0
  }
  const fixed = FIXED[need.rollForward || 'patch'] ?? PATCH_FIXED
  return installed.some((v) => {
    const have = parts(v)
    if (!have) {
      return false
    }
    // The first part that differs decides: a fixed one must not, a free one must be higher.
    const at = have.findIndex((n, i) => n !== want[i])
    return at < 0 || (at >= fixed && (have[at] ?? 0) > (want[at] ?? 0))
  })
}
