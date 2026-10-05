import type {
  PrefSpec,
  Settings,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { currentGame } from '../nav/currentGame.ts'

export function prefRaw(settings: Settings, spec: PrefSpec, game = currentGame()): unknown {
  if (spec.scope === 'game') {
    const block = settings.games?.[game]
    return block ? (block as unknown as Record<string, unknown>)[spec.key] : undefined
  }
  return (settings as unknown as Record<string, unknown>)[spec.key]
}

export function prefAsString(raw: unknown, spec: PrefSpec): string {
  if (raw === undefined || raw === null) {
    return spec.default
  }
  if (typeof raw === 'boolean') {
    return raw ? 'true' : 'false'
  }
  const s = String(raw)
  if (s === '' && spec.type !== 'string') {
    return spec.default
  }
  return s
}

export function prefAsBool(raw: unknown, spec: PrefSpec): boolean {
  return prefAsString(raw, spec) === 'true'
}

export function prefAsNumber(raw: unknown, spec: PrefSpec): number {
  const n = Number(prefAsString(raw, spec))
  return Number.isFinite(n) ? n : Number(spec.default)
}

export function specGameArg(spec: PrefSpec, game = currentGame()): string {
  return spec.scope === 'game' ? game : ''
}
