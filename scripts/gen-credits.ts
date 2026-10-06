#!/usr/bin/env bun
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { bundledArtwork } from './bundled-artwork.ts'

const LICENCE_ERROR_CHARS = 280
const PACKAGE_IN_PATH = /^.*node_modules\/((?:@[^/]+\/)?[^/]+)/
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
  { id: 'ISC', re: /permission to use, copy, modify, and(\/or)? distribute this software/i },
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

// Direct dependencies with their module directories, from one `go list -m`.
function directGoModules(): { path: string; dir: string }[] {
  const proc = Bun.spawnSync(
    ['go', 'list', '-m', '-f', '{{if not (or .Main .Indirect)}}{{.Path}}\t{{.Dir}}{{end}}', 'all'],
    { cwd: repoRoot, stdout: 'pipe', stderr: 'pipe' },
  )
  if (proc.exitCode !== 0) {
    throw new Error(`go list -m all: ${proc.stderr.toString()}`)
  }
  return proc.stdout
    .toString()
    .split('\n')
    .filter(Boolean)
    .map((line) => {
      const [path = '', dir = ''] = line.split('\t')
      if (!dir) {
        throw new Error(`go list -m ${path}: no Dir`)
      }
      return { path, dir }
    })
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
  if (!(path && path !== 'github.com/Rethunk-Tech/mortar')) {
    return
  }
  const name = version ? `${path}@${version}` : path
  if (seen.has(name)) {
    return
  }
  seen.add(name)
  const dir = listedDir && existsSync(listedDir) ? listedDir : ''
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

// Module lines (path, version, dir) of every package linked into each release target, one `go list -deps` per OS.
async function listGoDeps(): Promise<string[]> {
  const outputs = await Promise.all(
    releaseTargets.map(async (goos) => {
      // -e: main.go embeds frontend/dist, which this build has not produced yet on a clean checkout (CI); the module
      // list does not depend on it.
      const proc = Bun.spawn(
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
      const [out, err, code] = await Promise.all([
        new Response(proc.stdout).text(),
        new Response(proc.stderr).text(),
        proc.exited,
      ])
      if (code !== 0) {
        throw new Error(`go list -deps (${goos}): ${err}`)
      }
      return out
    }),
  )
  return outputs.flatMap((out) => out.split('\n'))
}

function goModuleNotices(lines: string[]): NoticeEntry[] {
  const seen = new Set<string>()
  const out: NoticeEntry[] = []
  const missing: string[] = []
  for (const line of lines) {
    appendGoNotice(line, seen, out, missing)
  }
  failMissing('Go modules', missing)
  return out
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

// The files in the app's bundle, from the bundler's module graph rather than the dependency tree, which also names
// build-only packages that never ship.
async function bundleInputs(): Promise<string[]> {
  const build = await Bun.build({
    entrypoints: [join(frontendRoot, 'src', 'main.tsx')],
    target: 'browser',
    metafile: true,
    throw: false,
  })
  if (!(build.success && build.metafile)) {
    throw new Error(`bundle graph: ${build.logs.map(String).join('\n')}`)
  }
  return Object.keys(build.metafile.inputs)
}

function npmNotices(inputs: string[]): NoticeEntry[] {
  // Keyed by folder, not name: the bundle can hold two versions of one package, each with its own notice.
  const dirs = new Map<string, string>()
  for (const input of inputs) {
    const m = PACKAGE_IN_PATH.exec(input)
    if (m?.[1]) {
      // The bundler reports inputs relative to the working directory.
      dirs.set(resolve(m[0]), m[1])
    }
  }
  const out = new Map<string, NoticeEntry>()
  const missing: string[] = []
  for (const [dir, name] of dirs) {
    const notice = npmNoticeFromDir(name, dir)
    if ('missing' in notice) {
      missing.push(notice.missing)
    } else {
      out.set(notice.name, notice)
    }
  }
  failMissing('npm packages', missing)
  return [...out.values()]
}

// The Go standard library is compiled into every build; its licence is the toolchain's own LICENSE.
function goStdlibNotice(goroot: string, version: string): NoticeEntry {
  const text = readFileSync(join(goroot, 'LICENSE'), 'utf8').trim()
  return {
    name: `Go standard library ${version}`,
    licence: classifyLicenceText(text),
    url: 'https://go.dev/',
    texts: [text],
  }
}

/** What the notices are built from: the Go toolchain, the linked Go modules and the bundled files. */
interface NoticeSources {
  goroot: string
  goVersion: string
  goDeps: string[]
  bundleInputs: string[]
}

// The Go graphs and the bundle graph are independent, so they load side by side.
async function gatherSources(): Promise<NoticeSources> {
  const env = Bun.spawn(['go', 'env', 'GOROOT', 'GOVERSION'], { cwd: repoRoot, stdout: 'pipe' })
  const [envOut, goDeps, inputs] = await Promise.all([
    new Response(env.stdout).text(),
    listGoDeps(),
    bundleInputs(),
  ])
  const [goroot = '', goVersion = ''] = envOut.trim().split('\n')
  return { goroot, goVersion, goDeps, bundleInputs: inputs }
}

function collectNotices(sources: NoticeSources): NoticeEntry[] {
  const entries = [
    goStdlibNotice(sources.goroot, sources.goVersion),
    ...goModuleNotices(sources.goDeps),
    ...npmNotices(sources.bundleInputs),
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
    'The Go standard library and the Go modules linked into Mortar on each released',
    'platform, the npm packages in the frontend bundle, and bundled artwork, with',
    'licence and NOTICE text from each.',
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
  const go = directGoModules().map(({ path, dir }) => ({
    name: path,
    licence: classifyLicenceText(readLicenceFile(dir)),
    url: goModuleURL(path),
  }))
  // Only what ships inside Mortar: SMAPI and the bridge are downloaded into a profile, never bundled.
  const art = bundledArtwork.map(({ notice: _notice, ...credit }) => credit)
  return [...art, ...npm, ...go]
}

async function main(): Promise<void> {
  Bun.spawnSync(['go', 'mod', 'download'], { cwd: repoRoot, stdout: 'pipe', stderr: 'pipe' })
  const entries = buildCredits()
  if (!isCreditList(entries)) {
    throw new Error('generated credits failed shape check')
  }
  const notices = collectNotices(await gatherSources())
  const outDir = join(frontendRoot, 'src', 'settings', 'generated')
  mkdirSync(outDir, { recursive: true })
  writeFileSync(join(outDir, 'credits.json'), `${JSON.stringify(entries, null, 2)}\n`)
  writeFileSync(join(repoRoot, 'THIRD_PARTY_NOTICES'), formatNotices(notices))
}

if (import.meta.main) {
  await main()
}

export { classifyLicenceText, collectNotices, type NoticeSources }
