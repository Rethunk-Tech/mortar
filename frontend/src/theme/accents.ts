export const accents = {
  sand: '#D6B17A',
  moss: '#93B86A',
  copper: '#D98C5F',
  sky: '#79AEDC',
  rose: '#E08A9B',
  lavender: '#A893DE',
  teal: '#5DB8AE',
  slate: '#94A7BC',
} as const

export type AccentName = keyof typeof accents

export const defaultAccent: AccentName = 'sand'
