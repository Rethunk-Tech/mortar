import init, { DecompressStream } from '../../vendor/brotli-dec-wasm/brotli_dec_wasm.js'

const MAX_ENCODED = 8192
const MAX_DECODED = 65_536
const VERSION = 3
const PART = /^\w+$/
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

// An entry is an object naming its source: {s: 'nexus', mod, file}, {s: 'github', repo, tag, asset} or
// {s: 'thunderstore', ns, name, version}.
function entry(e) {
  if (e && typeof e === 'object' && !Array.isArray(e)) {
    if (e.s === 'nexus' && Number.isInteger(e.mod) && e.mod > 0) {
      return { kind: 'nexus', mod: e.mod }
    }
    if (
      e.s === 'github' &&
      [e.repo, e.tag, e.asset].every((v) => typeof v === 'string' && v) &&
      REPO.test(e.repo)
    ) {
      return { kind: 'github', repo: e.repo, tag: e.tag, asset: e.asset }
    }
    if (e.s === 'thunderstore' && PART.test(e.ns) && PART.test(e.name)) {
      return { kind: 'thunderstore', ns: e.ns, name: e.name }
    }
  }
  throw new ShareError('bad')
}

// Payload is the text after "#": base64url(brotli(JSON [3, name, game, sourceKeys, entries])); only VERSION is read.
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
  if (Number.isInteger(json[0]) && json[0] > VERSION) {
    throw new ShareError('version')
  }
  const [version, name, game, sourceKeys, entries] = json
  if (
    version !== VERSION ||
    typeof name !== 'string' ||
    typeof game !== 'string' ||
    !game ||
    !sourceKeys ||
    typeof sourceKeys !== 'object' ||
    Array.isArray(sourceKeys) ||
    !Array.isArray(entries)
  ) {
    throw new ShareError('bad')
  }
  const out = entries.map(entry)
  for (const kind of ['nexus', 'thunderstore']) {
    if (out.some((e) => e.kind === kind) && typeof sourceKeys[kind] !== 'string') {
      throw new ShareError('bad')
    }
  }
  return { name, game, sourceKeys, entries: out }
}

export class ShareError extends Error {
  constructor(kind, options) {
    super(kind, options)
    this.kind = kind
  }
}
