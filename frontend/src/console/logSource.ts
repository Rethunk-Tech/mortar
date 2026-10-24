import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { HiddenLogSource } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'

// Unity names the object of some engine warnings on a later line of the message.
const SCENE_PATH = /Scene hierarchy path "([^"]+)"/
const UNITY_LOG = 'Unity Log'
// The bridge masks numbers the same way (LogTag.Mask), so a hidden line drops in the game too.
const NUMBER = /-?[0-9]+(?:[.,:][0-9]+)*/g

/** A message's first line with every number as "#": what one call keeps from line to line. */
export function mask(message: string): string {
  return (message.split('\n')[0] ?? '').replace(NUMBER, '#').trim()
}

/** Where a message (its header line and continuation lines) came from: the method the BepInEx bridge tagged it with
 * and its masked text, or for a line the engine logged itself, that masked line; with the object it names. Plugin lines
 * have none: their plugin is their source, which the mods filter already hides. */
export function sourceOf(group: readonly Entry[]): HiddenLogSource | null {
  const [header] = group
  if (!header) {
    return null
  }
  const label = header.message
  if (header.source) {
    return { source: header.source, text: mask(label), object: header.object ?? '', label }
  }
  if (header.mod !== UNITY_LOG || label.trim() === '') {
    return null
  }
  const object = group.map((e) => SCENE_PATH.exec(e.message)?.[1]).find((p) => p !== undefined)
  return { source: `unity:${mask(label)}`, text: '', object: object ?? '', label }
}

function covers(h: HiddenLogSource, src: HiddenLogSource): boolean {
  return (
    h.source === src.source &&
    (h.text ?? '') === (src.text ?? '') &&
    (!h.object || h.object === src.object)
  )
}

/** The log without the messages a hidden source covers, as if the game never wrote them. */
export function withoutHidden(entries: Entry[], hidden: readonly HiddenLogSource[]): Entry[] {
  if (hidden.length === 0) {
    return entries
  }
  const out: Entry[] = []
  let group: Entry[] = []
  const flush = () => {
    const src = sourceOf(group)
    if (!(src && hidden.some((h) => covers(h, src)))) {
      out.push(...group)
    }
    group = []
  }
  for (const entry of entries) {
    if (!entry.cont && group.length > 0) {
      flush()
    }
    group.push(entry)
  }
  flush()
  return out
}
