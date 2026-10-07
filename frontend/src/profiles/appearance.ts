const COLOR_HEX = {
  rose: '#e57373',
  orange: '#ffb74d',
  gold: '#ffd54f',
  lime: '#aed581',
  teal: '#4db6ac',
  sky: '#4fc3f7',
  violet: '#9575cd',
  pink: '#f06292',
} as const

export const PROFILE_COLORS = [
  'rose',
  'orange',
  'gold',
  'lime',
  'teal',
  'sky',
  'violet',
  'pink',
] as const

export const PROFILE_ICONS = [
  'sprout',
  'leaf',
  'wheat',
  'fish',
  'hammer',
  'pickaxe',
  'star',
  'heart',
  'mountain',
  'sun',
  'moon',
  'sparkles',
] as const

export type ProfileColor = (typeof PROFILE_COLORS)[number]
export type ProfileIcon = (typeof PROFILE_ICONS)[number]

export const MAX_NAME = 60
export const MAX_DESCRIPTION = 280

export function isProfileColor(value: string | undefined): value is ProfileColor {
  return PROFILE_COLORS.includes(value as ProfileColor)
}

export function isProfileIcon(value: string | undefined): value is ProfileIcon {
  return PROFILE_ICONS.includes(value as ProfileIcon)
}

export function colorHex(value: string | undefined): string | undefined {
  return isProfileColor(value) ? COLOR_HEX[value] : undefined
}

export function clipDescription(value: string): string {
  const trimmed = value.trim()
  return [...trimmed].slice(0, MAX_DESCRIPTION).join('')
}
