export interface Share {
  name: string
  game: string
  gameVersion: string
  sourceKeys: Record<string, string>
  entries: { kind: string; [field: string]: unknown }[]
}

export function decodeShare(hash: string, wasm?: BufferSource): Promise<Share>

export function newer(a: string, b: string): boolean

export class ShareError extends Error {
  kind: string
  constructor(kind: string, options?: ErrorOptions)
}
