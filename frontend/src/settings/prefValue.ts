import type {
  PrefSpec,
  Settings,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { currentGame } from '../nav/currentGame.ts'
import { specDefault } from './prefSpecs.ts'

export function prefRaw(settings: Settings, spec: PrefSpec, game = currentGame()): unknown {
  if (spec.scope === 'game') {
    const block = settings.games?.[game]
    return block ? (block as unknown as Record<string, unknown>)[spec.key] : undefined
  }
  return (settings as unknown as Record<string, unknown>)[spec.key]
}

export function prefAsString(raw: unknown, spec: PrefSpec, game = currentGame()): string {
  const fallback = specDefault(spec, game)
  if (raw === undefined || raw === null) {
    return fallback
  }
  if (typeof raw === 'boolean') {
    return raw ? 'true' : 'false'
  }
  const s = String(raw)
  if (s === '' && spec.type !== 'string') {
    return fallback
  }
  return s
}

export function prefAsBool(raw: unknown, spec: PrefSpec, game = currentGame()): boolean {
  return prefAsString(raw, spec, game) === 'true'
}

export function prefAsNumber(raw: unknown, spec: PrefSpec, game = currentGame()): number {
  const n = Number(prefAsString(raw, spec, game))
  return Number.isFinite(n) ? n : Number(specDefault(spec, game))
}

export function specGameArg(spec: PrefSpec, game = currentGame()): string {
  return spec.scope === 'game' ? game : ''
}
