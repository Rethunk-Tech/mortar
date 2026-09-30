export const accents = {
  sand: '#D6B17A',
  moss: '#93B86A',
  copper: '#D98C5F',
  sky: '#79AEDC',
} as const

export type AccentName = keyof typeof accents

export const defaultAccent: AccentName = 'sand'
