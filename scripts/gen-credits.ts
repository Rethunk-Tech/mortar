#!/usr/bin/env bun
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { bundledArtwork } from './bundled-artwork.ts'

const LICENCE_ERROR_CHARS = 280
const here = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(here, '..')
const frontendRoot = join(repoRoot, 'frontend')
const licenceFileName = /^(license|licence|copying)(\..+)?$/i
const noticeFileName = /^notice(\..+)?$/i

/** Shipped packages with neither a licence file nor a declared licence. Text is the upstream licence. */
const NPM_LICENCE_ALLOWLIST: Record<string, { spdx: string; text: string; url?: string }> = {}
const GO_LICENCE_ALLOWLIST: Record<string, { spdx: string; text: string; url?: string }> = {}

const LICENCE_RULES: { id: string; re: RegExp; extra?: RegExp }[] = [
  { id: 'AGPL-3.0', re: /GNU AFFERO GENERAL PUBLIC LICENSE/i, extra: /Version 3/i },
  { id: 'LGPL-3.0', re: /GNU LESSER GENERAL PUBLIC LICENSE/i, extra: /Version 3/i },
  { id: 'LGPL-2.1', re: /GNU LESSER GENERAL PUBLIC LICENSE/i, extra: /Version 2\.1/i },
  { id: 'GPL-3.0', re: /GNU GENERAL PUBLIC LICENSE/i, extra: /Version 3/i },
  { id: 'MPL-2.0', re: /mozilla public license/i, extra: /2\.0/i },
  { id: 'Apache-2.0', re: /apache license/i, extra: /version 2\.0/i },
  { id: 'OFL-1.1', re: /sil open font license/i },
  { id: 'CC-BY-SA-4.0', re: /creative commons/i, extra: /attribution-sharealike 4\.0/i },
  { id: 'MIT', re: /the mit license/i },
  { id: 'MIT', re: /permission is hereby granted, free of charge/i },
  { id: 'ISC', re: /permission to use, copy, modify, and\/or distribute this software/i },
  {
    id: 'BSD-3-Clause',
    re: /redistribution and use in source and binary forms/i,
    extra: /neither the name/i,
  },
  { id: 'BSD-3-Clause', re: /bsd 3-clause/i },
  { id: 'BSD-2-Clause', re: /redistribution and use in source and binary forms/i },
  { id: 'Unlicense', re: /unlicense/i },
]

function asRecord(v: unknown): Record<string, unknown> {
  return v !== null && typeof v === 'object' && !Array.isArray(v)
    ? (v as Record<string, unknown>)
    : {}
}

function stringField(v: unknown, key: string): string {
  const n = asRecord(v)[key]
  return typeof n === 'string' ? n : ''
}

function readJSON(path: string): unknown {
  return JSON.parse(readFileSync(path, 'utf8'))
}

function npmLicence(pkgJson: unknown): string {
  const rec = asRecord(pkgJson)
  if (typeof rec.license === 'string' && rec.license.trim()) {
    return rec.license.trim()
  }
  const { licenses } = rec
  if (Array.isArray(licenses)) {
    const parts = licenses
      .map((item) => (typeof item === 'string' ? item : stringField(item, 'type')))
      .filter(Boolean)
    if (parts.length > 0) {
      return parts.join(' OR ')
    }
  }
  throw new Error(`no licence in package.json for ${stringField(pkgJson, 'name') || '?'}`)
}

function declaredLicence(pkgJson: unknown): string {
  try {
    return npmLicence(pkgJson)
  } catch {
    return ''
  }
}

function githubURL(raw: string): string {
  let u = raw.trim()
  u = u.replace(/^git\+/, '').replace(/^ssh:\/\/git@/, 'https://')
  u = u.replace(/^git@github\.com:/, 'https://github.com/')
  u = u.replace(/\.git$/, '')
  if (u.startsWith('github:')) {
    u = `https://github.com/${u.slice('github:'.length)}`
  }
  if (!/^https?:\/\//.test(u)) {
    u = `https://github.com/${u}`
  }
  return u
}

function npmHomepage(name: string, pkgJson: unknown): string {
  const rec = asRecord(pkgJson)
  if (typeof rec.homepage === 'string' && rec.homepage.trim()) {
    return rec.homepage.trim().replace(/#.*$/, '')
  }
  const { repository } = rec
  if (typeof repository === 'string' && repository.trim()) {
    return githubURL(repository)
  }
  if (repository && typeof repository === 'object') {
    const url = stringField(repository, 'url')
    if (url) {
      return githubURL(url)
    }
  }
  return `https://www.npmjs.com/package/${encodeURIComponent(name)}`
}

function firstRequireBlock(goMod: string): string[] {
  const m = /require\s*\(([^)]*)\)/.exec(goMod)
  if (!m) {
    throw new Error('go.mod: no require block')
  }
  const mods: string[] = []
  for (const line of (m[1] ?? '').split('\n')) {
    const trimmed = line.replace(/\/\/.*$/, '').trim()
    if (trimmed) {
      const [path] = trimmed.split(/\s+/)
      if (path) {
        mods.push(path)
      }
    }
  }
  if (mods.length === 0) {
    throw new Error('go.mod: empty first require block')
  }
  return mods
}

function goListDir(modulePath: string): string {
  const proc = Bun.spawnSync(['go', 'list', '-m', '-json', modulePath], {
    cwd: repoRoot,
    stdout: 'pipe',
    stderr: 'pipe',
  })
  if (proc.exitCode !== 0) {
    throw new Error(`go list ${modulePath}: ${proc.stderr.toString()}`)
  }
  const rec = asRecord(JSON.parse(proc.stdout.toString()))
  const dir = typeof rec.Dir === 'string' ? rec.Dir : ''
  if (!dir) {
    throw new Error(`go list ${modulePath}: no Dir`)
  }
  return dir
}

function goModuleURL(modulePath: string): string {
  if (modulePath.startsWith('github.com/')) {
    const parts = modulePath.split('/')
    return `https://github.com/${parts[1] ?? ''}/${parts[2] ?? ''}`
  }
  return `https://pkg.go.dev/${modulePath}`
}

function readLicenceFile(dir: string): string {
  let names: string[]
  try {
    names = readdirSync(dir)
  } catch (err) {
    throw new Error(`cannot read ${dir}`, { cause: err })
  }
  const hit = names.find((n) => licenceFileName.test(n))
  if (!hit) {
    throw new Error(`no LICENSE file in ${dir}`)
  }
  return readFileSync(join(dir, hit), 'utf8')
}

function classifyLicenceText(text: string): string {
  const hit = LICENCE_RULES.find(
    (rule) => rule.re.test(text) && (!rule.extra || rule.extra.test(text)),
  )
  if (!hit) {
    throw new Error(`unrecognised licence text:\n${text.slice(0, LICENCE_ERROR_CHARS)}`)
  }
  return hit.id
}

interface CreditEntry {
  name: string
  licence: string
  url: string
}

interface NoticeEntry {
  name: string
  licence: string
  url: string
  texts: string[]
}

function isCreditList(v: unknown): v is CreditEntry[] {
  if (!Array.isArray(v) || v.length === 0) {
    return false
  }
  return v.every((item) => {
    const rec = asRecord(item)
    return (
      typeof rec.name === 'string' &&
      rec.name.length > 0 &&
      typeof rec.licence === 'string' &&
      rec.licence.length > 0 &&
      typeof rec.url === 'string' &&
      /^https?:\/\//.test(rec.url)
    )
  })
}

function dirTexts(dir: string): { licence: string; texts: string[] } | null {
  let names: string[]
  try {
    names = readdirSync(dir)
  } catch {
    return null
  }
  const licenceHits = names
    .filter((n) => licenceFileName.test(n))
    .sort((a, b) => a.localeCompare(b))
  const noticeHits = names.filter((n) => noticeFileName.test(n)).sort((a, b) => a.localeCompare(b))
  const texts = [...licenceHits, ...noticeHits].map((n) => readFileSync(join(dir, n), 'utf8'))
  if (texts.length === 0) {
    return null
  }
  let licence = 'see text'
  if (licenceHits[0]) {
    try {
      licence = classifyLicenceText(readFileSync(join(dir, licenceHits[0]), 'utf8'))
    } catch {
      licence = 'see text'
    }
  }
  return { licence, texts }
}

function goModuleDir(path: string, version: string, listedDir: string): string {
  if (listedDir && existsSync(listedDir)) {
    return listedDir
  }
  const spec = version ? `${path}@${version}` : path
  const proc = Bun.spawnSync(['go', 'list', '-m', '-f', '{{.Dir}}', spec], {
    cwd: repoRoot,
    stdout: 'pipe',
    stderr: 'pipe',
  })
  return proc.exitCode === 0 ? proc.stdout.toString().trim() : ''
}

function failMissing(kind: string, missing: string[]): void {
  if (missing.length > 0) {
    throw new Error(
      `no licence file or declared licence for ${kind}:\n${missing.sort((a, b) => a.localeCompare(b)).join('\n')}`,
    )
  }
}

function appendGoNotice(
  line: string,
  seen: Set<string>,
  out: NoticeEntry[],
  missing: string[],
): void {
  const trimmed = line.trim()
  const [path, version, listedDir] = trimmed ? trimmed.split('\t') : []
  if (!(path && path !== 'github.com/Rethunk-AI/mortar')) {
    return
  }
  const name = version ? `${path}@${version}` : path
  if (seen.has(name)) {
    return
  }
  seen.add(name)
  const dir = goModuleDir(path, version ?? '', listedDir ?? '')
  const found = dir ? dirTexts(dir) : null
  const allowed = GO_LICENCE_ALLOWLIST[path]
  if (found || allowed) {
    out.push({
      name,
      licence: found?.licence ?? allowed?.spdx ?? 'unknown',
      url: allowed?.url ?? goModuleURL(path),
      texts: found?.texts ?? (allowed ? [allowed.text] : []),
    })
    return
  }
  missing.push(name)
}

// Each released build links its own modules (keyring and notifications differ by OS), so the notices cover the
// modules linked into every target Mortar ships.
const releaseTargets = ['linux', 'windows']

function goModuleNotices(): NoticeEntry[] {
  const seen = new Set<string>()
  const out: NoticeEntry[] = []
  const missing: string[] = []
  for (const goos of releaseTargets) {
    const proc = Bun.spawnSync(
      // -e: main.go embeds frontend/dist, which this build has not produced yet on a clean checkout (CI); the module
      // list does not depend on it.
      [
        'go',
        'list',
        '-e',
        '-deps',
        '-f',
        '{{with .Module}}{{.Path}}\t{{.Version}}\t{{.Dir}}{{end}}',
        '.',
      ],
      { cwd: repoRoot, env: { ...process.env, GOOS: goos }, stdout: 'pipe', stderr: 'pipe' },
    )
    if (proc.exitCode !== 0) {
      throw new Error(`go list -deps (${goos}): ${proc.stderr.toString()}`)
    }
    for (const line of proc.stdout.toString().split('\n')) {
      appendGoNotice(line, seen, out, missing)
    }
  }
  failMissing('Go modules', missing)
  return out
}

function resolveNpmDir(name: string, fromDir: string): string | null {
  try {
    return dirname(Bun.resolveSync(`${name}/package.json`, fromDir))
  } catch {
    return null
  }
}

function npmNoticeFromDir(name: string, dir: string): NoticeEntry | { missing: string } {
  const meta = readJSON(join(dir, 'package.json'))
  const pkgName = stringField(meta, 'name') || name
  const version = stringField(meta, 'version')
  const key = version ? `${pkgName}@${version}` : pkgName
  const found = dirTexts(dir)
  const declared = declaredLicence(meta)
  const allowed = NPM_LICENCE_ALLOWLIST[pkgName]
  if (!(found || declared || allowed)) {
    return { missing: key }
  }
  const licence =
    found?.licence && found.licence !== 'see text'
      ? found.licence
      : declared || allowed?.spdx || 'see text'
  return {
    name: key,
    licence,
    url: allowed?.url ?? npmHomepage(pkgName, meta),
    texts: found?.texts ?? (allowed ? [allowed.text] : []),
  }
}

function visitNpm(
  item: { name: string; from: string; optional: boolean },
  ctx: {
    seen: Set<string>
    queue: { name: string; from: string; optional: boolean }[]
    out: NoticeEntry[]
    missing: string[]
  },
): void {
  const dir = resolveNpmDir(item.name, item.from)
  if (!dir) {
    if (item.optional || item.from !== frontendRoot) {
      return
    }
    throw new Error(`npm package not installed: ${item.name}`)
  }
  const meta = readJSON(join(dir, 'package.json'))
  const pkgName = stringField(meta, 'name') || item.name
  const version = stringField(meta, 'version')
  const key = version ? `${pkgName}@${version}` : pkgName
  if (ctx.seen.has(key)) {
    return
  }
  ctx.seen.add(key)
  const notice = npmNoticeFromDir(item.name, dir)
  if ('missing' in notice) {
    ctx.missing.push(notice.missing)
  } else {
    ctx.out.push(notice)
  }
  const rec = asRecord(meta)
  for (const dep of Object.keys(asRecord(rec.dependencies))) {
    ctx.queue.push({ name: dep, from: dir, optional: false })
  }
  for (const dep of Object.keys(asRecord(rec.optionalDependencies))) {
    ctx.queue.push({ name: dep, from: dir, optional: true })
  }
}

function npmNotices(): NoticeEntry[] {
  const pkg = asRecord(readJSON(join(frontendRoot, 'package.json')))
  const queue = Object.keys(asRecord(pkg.dependencies)).map((name) => ({
    name,
    from: frontendRoot,
    optional: false,
  }))
  const seen = new Set<string>()
  const out: NoticeEntry[] = []
  const missing: string[] = []
  while (queue.length > 0) {
    const item = queue.shift()
    if (item) {
      visitNpm(item, { seen, queue, out, missing })
    }
  }
  failMissing('npm packages', missing)
  return out
}

function collectNotices(): NoticeEntry[] {
  const entries = [
    ...goModuleNotices(),
    ...npmNotices(),
    ...bundledArtwork.map(({ notice, ...credit }) => ({ ...credit, texts: [notice] })),
  ]
  entries.sort((a, b) => a.name.localeCompare(b.name))
  return entries
}

function formatNotices(entries: NoticeEntry[]): string {
  const blocks = entries.map((e) => {
    const header = `${e.name}\nlicence: ${e.licence}\n${e.url}`
    const body = e.texts.join('\n\n').trim()
    return body ? `${header}\n\n${body}\n` : `${header}\n`
  })
  return [
    'Third-party notices',
    '',
    'Go modules compiled into Mortar, the frontend runtime npm closure (direct',
    'dependencies of frontend/package.json and their installed dependency trees),',
    'with licence and NOTICE text from each package directory, and bundled artwork.',
    '',
    '================================================================================',
    '',
    blocks.join(
      '\n--------------------------------------------------------------------------------\n\n',
    ),
    '',
  ].join('\n')
}

function buildCredits(): CreditEntry[] {
  const pkg = asRecord(readJSON(join(frontendRoot, 'package.json')))
  const npm = Object.keys(asRecord(pkg.dependencies))
    .sort((a, b) => a.localeCompare(b))
    .map((name) => {
      const meta = readJSON(join(frontendRoot, 'node_modules', ...name.split('/'), 'package.json'))
      return { name, licence: npmLicence(meta), url: npmHomepage(name, meta) }
    })
  const go = firstRequireBlock(readFileSync(join(repoRoot, 'go.mod'), 'utf8')).map((mod) => ({
    name: mod,
    licence: classifyLicenceText(readLicenceFile(goListDir(mod))),
    url: goModuleURL(mod),
  }))
  // Only what ships inside Mortar: SMAPI and the bridge are downloaded into a profile, never bundled.
  const art = bundledArtwork.map(({ notice: _notice, ...credit }) => credit)
  return [...art, ...npm, ...go]
}

function main(): void {
  Bun.spawnSync(['go', 'mod', 'download'], { cwd: repoRoot, stdout: 'pipe', stderr: 'pipe' })
  const entries = buildCredits()
  if (!isCreditList(entries)) {
    throw new Error('generated credits failed shape check')
  }
  const notices = collectNotices()
  const outDir = join(frontendRoot, 'src', 'settings', 'generated')
  mkdirSync(outDir, { recursive: true })
  writeFileSync(join(outDir, 'credits.json'), `${JSON.stringify(entries, null, 2)}\n`)
  writeFileSync(join(repoRoot, 'THIRD_PARTY_NOTICES'), formatNotices(notices))
}

if (import.meta.main) {
  main()
}

export { classifyLicenceText, collectNotices }
