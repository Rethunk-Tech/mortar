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
const MODDB_ID = /^\d{1,9}-\d{1,9}$/
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
// {s: 'thunderstore', ns, name, version}, {s: 'curseforge', mod: project, file}, {s: 'vsmoddb', name: 'modid-assetid',
// file, version}, {s: 'itch', name: 'user/game'} or
// {s: 'patreon', name: post id}, with optional page facts.
function entry(e) {
  return { ...source(e), ...facts(e) }
}

const positive = (n) => Number.isInteger(n) && n > 0
const matches = (v, re) => typeof v === 'string' && re.test(v)

// Each source's reader returns its entry, or null when the object does not hold a well-formed one.
const READERS = {
  nexus: (e) => (positive(e.mod) ? { kind: 'nexus', mod: e.mod } : null),
  github: (e) =>
    [e.repo, e.tag, e.asset].every((v) => typeof v === 'string' && v) && REPO.test(e.repo)
      ? { kind: 'github', repo: e.repo, tag: e.tag, asset: e.asset }
      : null,
  curseforge: (e) =>
    positive(e.mod) && positive(e.file) ? { kind: 'curseforge', project: e.mod } : null,
  vsmoddb: (e) =>
    matches(e.name, MODDB_ID) && positive(e.file) ? { kind: 'vsmoddb', id: e.name } : null,
  itch: (e) => (matches(e.name, ITCH_PAGE) ? { kind: 'itch', page: e.name } : null),
  patreon: (e) => (matches(e.name, POST_ID) ? { kind: 'patreon', post: e.name } : null),
  thunderstore: (e) => {
    if (!(PART.test(e.ns) && PART.test(e.name))) {
      return null
    }
    const out = { kind: 'thunderstore', ns: e.ns, name: e.name }
    if (matches(e.version, PACKAGE_VERSION)) {
      out.version = e.version
    }
    return out
  },
}

function source(e) {
  const read =
    e && typeof e === 'object' && !Array.isArray(e) && Object.hasOwn(READERS, e.s)
      ? READERS[e.s]
      : null
  const out = read?.(e)
  if (!out) {
    throw new ShareError('bad')
  }
  return out
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
  for (const kind of ['nexus', 'thunderstore', 'curseforge', 'vsmoddb']) {
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
