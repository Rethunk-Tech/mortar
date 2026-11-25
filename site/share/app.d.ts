export interface ShortShare {
  game: string
  code: string
  version: number
}

export function shortFromPath(pathname: string): ShortShare | null

export function shortAPIPath(short: ShortShare): string

export function shortAppLink(short: ShortShare): string
