const TIP_IDS = ['mods', 'console', 'share'] as const

export type TipId = (typeof TIP_IDS)[number]

export function tipVisible(seen: readonly string[] | null | undefined, id: TipId): boolean {
  return !(seen ?? []).includes(id)
}
