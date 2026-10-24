import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { HiddenLogSource } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'

// Unity names the object of some engine warnings on a later line of the message.
const SCENE_PATH = /Scene hierarchy path "([^"]+)"/
const UNITY_LOG = 'Unity Log'

/** Where a message (its header line and continuation lines) came from: the call site the BepInEx bridge tagged it
 * with, or for a line the engine logged itself, its first line and the object it names. Plugin lines have none: their
 * plugin is their source, which the mods filter already hides. */
export function sourceOf(group: readonly Entry[]): HiddenLogSource | null {
  const [header] = group
  if (!header) {
    return null
  }
  if (header.source) {
    return { source: header.source, object: header.object ?? '', label: header.message }
  }
  if (header.mod !== UNITY_LOG || header.message.trim() === '') {
    return null
  }
  const object = group.map((e) => SCENE_PATH.exec(e.message)?.[1]).find((p) => p !== undefined)
  return { source: `unity:${header.message}`, object: object ?? '', label: header.message }
}

/** Whether a hidden source covers the message: the same source, about any object or the same one. */
export function hides(hidden: readonly HiddenLogSource[], group: readonly Entry[]): boolean {
  if (hidden.length === 0) {
    return false
  }
  const src = sourceOf(group)
  return (
    src !== null &&
    hidden.some((h) => h.source === src.source && (!h.object || h.object === src.object))
  )
}
