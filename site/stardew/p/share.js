import init, { DecompressStream } from '../../vendor/brotli-dec-wasm/brotli_dec_wasm.js'

const MAX_ENCODED = 8192
const MAX_DECODED = 65_536
const VERSION = 1
const REPO = /^[\w.-]+\/[\w.-]+$/
const HASH = /^#/
const B64URL = /^[\w-]+$/
const DASH = /-/g
const UNDERSCORE = /_/g

let ready
const wasmUrl = new URL('../../vendor/brotli-dec-wasm/brotli_dec_wasm_bg.wasm', import.meta.url)

function inflate(bytes, wasm) {
  ready ??= init({ module_or_path: wasm ?? wasmUrl })
  return ready.then(() => {
    const r = new DecompressStream().decompress(bytes, MAX_DECODED + 1)
    // 1 = ResultSuccess; anything else is truncated input or output over the cap.
    if (r.code !== 1 || r.buf.length > MAX_DECODED) {
      throw new ShareError('bad')
    }
    return r.buf
  })
}

function entry(e) {
  if (Array.isArray(e) && Number.isInteger(e[0]) && e[0] > 0) {
    return { kind: 'nexus', mod: e[0] }
  }
  if (typeof e === 'string') {
    const at = e.indexOf('@')
    const slash = e.lastIndexOf('/')
    const repo = e.slice(0, at)
    if (at > 0 && slash > at + 1 && REPO.test(repo)) {
      return { kind: 'github', repo, tag: e.slice(at + 1, slash), asset: e.slice(slash + 1) }
    }
  }
  throw new ShareError('bad')
}

// Payload is the text after "#": base64url(brotli(JSON [1, name, entries])).
export async function decodeShare(hash, wasm) {
  const payload = hash.replace(HASH, '')
  if (!payload) {
    throw new ShareError('missing')
  }
  if (payload.length > MAX_ENCODED || !B64URL.test(payload)) {
    throw new ShareError('bad')
  }
  const b64 = payload.replace(DASH, '+').replace(UNDERSCORE, '/')
  const bytes = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0))
  let json
  try {
    json = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(await inflate(bytes, wasm)))
  } catch (err) {
    throw new ShareError('bad', { cause: err })
  }
  if (!Array.isArray(json) || json.length === 0) {
    throw new ShareError('bad')
  }
  if (json[0] !== VERSION) {
    throw new ShareError(Number.isInteger(json[0]) ? 'version' : 'bad')
  }
  const [, name, entries] = json
  if (typeof name !== 'string' || !Array.isArray(entries)) {
    throw new ShareError('bad')
  }
  return { name, entries: entries.map(entry) }
}

export class ShareError extends Error {
  constructor(kind, options) {
    super(kind, options)
    this.kind = kind
  }
}
