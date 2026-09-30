import { type AccentName, accents } from '../theme/accents.ts'

export function isAccent(value: string): value is AccentName {
  return value in accents
}
