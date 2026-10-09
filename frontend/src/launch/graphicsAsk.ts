import type { Graphics } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'

interface GraphicsOption {
  id: string
  label: string
  recommended: boolean
}

/** The catalog's choices with the recommended one first, so the dialog leads with it. */
function graphicsOptions(graphics: Graphics): GraphicsOption[] {
  const all = (graphics.choices ?? []).map((c) => ({
    id: c.id,
    label: c.label,
    recommended: c.id === graphics.recommended,
  }))
  return [...all.filter((o) => o.recommended), ...all.filter((o) => !o.recommended)]
}

/** Where an answer is written: always the game's setting, and also the profile's override when it has one, since
 * that override would otherwise keep winning. */
function answerTargets(overrides: Record<string, string> | undefined): {
  profileOverride: boolean
} {
  return { profileOverride: overrides !== undefined && Object.hasOwn(overrides, 'graphicsApi') }
}

export { answerTargets, graphicsOptions }
