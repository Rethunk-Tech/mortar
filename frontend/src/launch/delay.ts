export const delay = (ms: number) => new Promise<void>((resolve) => globalThis.setTimeout(resolve, ms))
