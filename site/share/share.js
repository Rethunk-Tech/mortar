import init, { DecompressStream } from '../vendor/brotli-dec-wasm/brotli_dec_wasm.js'

const MAX_ENCODED = 8192
const MAX_DECODED = 65_536
const VERSION = 3
// PARTS is the payload array's length with its optional game version.
const PARTS = 6
const PART = /^\w+$/
const REPO = /^[\w.-]+\/[\w.-]+$/
const GAME_VERSION = /^\d{1,9}(\.\d{1,9}){0,3}$/
const PACKAGE_VERSION = /^\d+\.\d+\.\d+$/
const ITCH_PAGE = /^[a-z0-9][\w-]{0,38}\/[A-Za-z0-9][\w-]{0,99}$/
const POST_ID = /^\d{1,12}$/
const HASH = /^#/
const B64URL = /^[\w-]+$/
const DASH = /-/g
const UNDERSCORE = /_/g

let ready
const wasmUrl = new URL('../vendor/brotli-dec-wasm/brotli_dec_wasm_bg.wasm', import.meta.url)

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

// facts are what the page shows beside a source: kb, the size rounded to two digits, and min, the oldest game version
// the mod runs on. Either may be absent.
function facts(e) {
  const out = {}
  if (Number.isInteger(e.kb) && e.kb > 0) {
    out.kb = e.kb
  }
  if (typeof e.min === 'string' && GAME_VERSION.test(e.min)) {
    out.min = e.min
  }
  return out
}

// An entry is an object naming its source: {s: 'nexus', mod, file}, {s: 'github', repo, tag, asset} or
// {s: 'thunderstore', ns, name, version}, {s: 'curseforge', mod: project, file}, {s: 'itch', name: 'user/game'} or
// {s: 'patreon', name: post id}, with optional page facts.
function entry(e) {
  return { ...source(e), ...facts(e) }
}

function source(e) {
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
    if (
      e.s === 'curseforge' &&
      Number.isInteger(e.mod) &&
      e.mod > 0 &&
      Number.isInteger(e.file) &&
      e.file > 0
    ) {
      return { kind: 'curseforge', project: e.mod }
    }
    if (e.s === 'itch' && typeof e.name === 'string' && ITCH_PAGE.test(e.name)) {
      return { kind: 'itch', page: e.name }
    }
    if (e.s === 'patreon' && typeof e.name === 'string' && POST_ID.test(e.name)) {
      return { kind: 'patreon', post: e.name }
    }
    if (e.s === 'thunderstore' && PART.test(e.ns) && PART.test(e.name)) {
      const out = { kind: 'thunderstore', ns: e.ns, name: e.name }
      if (typeof e.version === 'string' && PACKAGE_VERSION.test(e.version)) {
        out.version = e.version
      }
      return out
    }
  }
  throw new ShareError('bad')
}

// Payload is the text after "#": base64url(brotli(JSON [3, name, game, sourceKeys, entries, gameVersion?])); only
// VERSION is read, and the game version the profile was shared from is optional.
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
  const [version, name, game, sourceKeys, entries, gameVersion = ''] = json
  if (
    version !== VERSION ||
    json.length > PARTS ||
    typeof gameVersion !== 'string' ||
    (gameVersion !== '' && !GAME_VERSION.test(gameVersion)) ||
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
  for (const kind of ['nexus', 'thunderstore', 'curseforge']) {
    if (out.some((e) => e.kind === kind) && typeof sourceKeys[kind] !== 'string') {
      throw new ShareError('bad')
    }
  }
  return { name, game, gameVersion, sourceKeys, entries: out }
}

// newer reports whether version a is above b, comparing dot-separated numbers; a missing part counts as 0.
export function newer(a, b) {
  const x = a.split('.').map(Number)
  const y = b.split('.').map(Number)
  for (let i = 0; i < Math.max(x.length, y.length); i++) {
    if ((x[i] ?? 0) !== (y[i] ?? 0)) {
      return (x[i] ?? 0) > (y[i] ?? 0)
    }
  }
  return false
}

export class ShareError extends Error {
  constructor(kind, options) {
    super(kind, options)
    this.kind = kind
  }
}
