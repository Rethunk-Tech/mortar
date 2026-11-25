export function originLine(
  origin: string | undefined,
  copyOf: string | undefined,
  labels: { link: string; mortar: string; gameMods: string; copy: (name: string) => string },
): string | undefined {
  switch (origin) {
    case 'link':
      return labels.link
    case 'mortar':
      return labels.mortar
    case 'game-mods':
      return labels.gameMods
    case 'copy':
      return copyOf ? labels.copy(copyOf) : undefined
    default:
      return undefined
  }
}

export function knownCount(n: number | undefined, label: string): string | undefined {
  return n === undefined || n <= 0 ? undefined : label
}

export function joinSummary(parts: Array<string | undefined>): string {
  return parts.filter((p): p is string => p !== undefined && p !== '').join(' · ')
}

// A profile's problems as the header chip and the sidebar badge count them: the missing mods are part of the total,
// though the badge store keeps them in their own field.
export const problemTotal = (
  badge: { problems: number; missing: number } | undefined,
): number | undefined => badge && badge.problems + badge.missing
