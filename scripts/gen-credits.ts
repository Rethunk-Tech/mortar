#!/usr/bin/env bun
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const LICENCE_ERROR_CHARS = 280

const here = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(here, '..')
const frontendRoot = join(repoRoot, 'frontend')
const licenceFileName = /^(license|licence|copying)(\..+)?$/i
const noticeFileName = /^notice(\..+)?$/i

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
    const owner = parts[1] ?? ''
    const repo = parts[2] ?? ''
    return `https://github.com/${owner}/${repo}`
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
  const body = text
  if (/GNU AFFERO GENERAL PUBLIC LICENSE/i.test(body) && /Version 3/i.test(body)) {
    return 'AGPL-3.0'
  }
  if (/GNU LESSER GENERAL PUBLIC LICENSE/i.test(body) && /Version 3/i.test(body)) {
    return 'LGPL-3.0'
  }
  if (/GNU LESSER GENERAL PUBLIC LICENSE/i.test(body) && /Version 2\.1/i.test(body)) {
    return 'LGPL-2.1'
  }
  if (/GNU GENERAL PUBLIC LICENSE/i.test(body) && /Version 3/i.test(body)) {
    return 'GPL-3.0'
  }
  if (/mozilla public license/i.test(body) && /2\.0/i.test(body)) {
    return 'MPL-2.0'
  }
  if (/apache license/i.test(body) && /version 2\.0/i.test(body)) {
    return 'Apache-2.0'
  }
  if (/sil open font license/i.test(body)) {
    return 'OFL-1.1'
  }
  if (/creative commons/i.test(body) && /attribution-sharealike 4\.0/i.test(body)) {
    return 'CC-BY-SA-4.0'
  }
  if (
    /the mit license/i.test(body) ||
    (/permission is hereby granted, free of charge/i.test(body) && /\bMIT\b/.test(body))
  ) {
    return 'MIT'
  }
  if (/permission is hereby granted, free of charge/i.test(body)) {
    return 'MIT'
  }
  if (/permission to use, copy, modify, and\/or distribute this software/i.test(body)) {
    return 'ISC'
  }
  if (
    /redistribution and use in source and binary forms/i.test(body) &&
    /neither the name/i.test(body)
  ) {
    return 'BSD-3-Clause'
  }
  if (/bsd 3-clause/i.test(body)) {
    return 'BSD-3-Clause'
  }
  if (/redistribution and use in source and binary forms/i.test(body)) {
    return 'BSD-2-Clause'
  }
  if (/unlicense/i.test(body)) {
    return 'Unlicense'
  }
  throw new Error(`unrecognised licence text:\n${body.slice(0, LICENCE_ERROR_CHARS)}`)
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
  const licenceHits = names.filter((n) => licenceFileName.test(n)).sort()
  const noticeHits = names.filter((n) => noticeFileName.test(n)).sort()
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

function parseGoListJSONStream(stdout: string): Record<string, unknown>[] {
  const trimmed = stdout.trim()
  if (!trimmed) {
    return []
  }
  return trimmed.split(/\n}\s*\n\{/).map((chunk, i, arr) => {
    let body = chunk
    if (i > 0) {
      body = `{${body}`
    }
    if (i < arr.length - 1) {
      body = `${body}\n}`
    }
    return asRecord(JSON.parse(body))
  })
}

function goModuleNotices(): NoticeEntry[] {
  Bun.spawnSync(['go', 'mod', 'download'], {
    cwd: repoRoot,
    stdout: 'pipe',
    stderr: 'pipe',
  })
  const proc = Bun.spawnSync(['go', 'list', '-m', '-json', 'all'], {
    cwd: repoRoot,
    stdout: 'pipe',
    stderr: 'pipe',
  })
  if (proc.exitCode !== 0) {
    throw new Error(`go list -m all: ${proc.stderr.toString()}`)
  }
  const recs = parseGoListJSONStream(proc.stdout.toString())
  const out: NoticeEntry[] = []
  for (const rec of recs) {
    const replace = asRecord(rec.Replace)
    const path = stringField(replace, 'Path') || stringField(rec, 'Path')
    const version = stringField(replace, 'Version') || stringField(rec, 'Version')
    const dir = stringField(replace, 'Dir') || stringField(rec, 'Dir')
    if (path && path !== 'github.com/Rethunk-AI/mortar') {
      const found = dir ? dirTexts(dir) : null
      const name = version ? `${path}@${version}` : path
      out.push({
        name,
        licence: found?.licence ?? 'unknown',
        url: goModuleURL(path),
        texts: found?.texts ?? [`(no LICENSE or NOTICE in module directory for ${path})\n`],
      })
    }
  }
  return out
}

function parseBunLockPackages(lockText: string): { name: string; spec: string }[] {
  const start = lockText.indexOf('"packages"')
  if (start < 0) {
    throw new Error('bun.lock: no packages table')
  }
  const body = lockText.slice(start)
  const pkgs: { name: string; spec: string }[] = []
  const re = /^\s+"([^"]+)": \["([^"]+)"/gm
  for (const m of body.matchAll(re)) {
    const name = m[1] ?? ''
    const spec = m[2] ?? ''
    if (name && spec && name !== 'mortar' && name !== 'mortar-frontend') {
      pkgs.push({ name, spec })
    }
  }
  if (pkgs.length === 0) {
    throw new Error('bun.lock: empty packages table')
  }
  return pkgs
}

function npmPackageDir(name: string): string | null {
  const candidates = [
    join(frontendRoot, 'node_modules', ...name.split('/')),
    join(repoRoot, 'node_modules', ...name.split('/')),
  ]
  for (const dir of candidates) {
    if (existsSync(join(dir, 'package.json'))) {
      return dir
    }
  }
  for (const base of [frontendRoot, repoRoot]) {
    try {
      return dirname(Bun.resolveSync(`${name}/package.json`, base))
    } catch {
      // try the other base
    }
  }
  return null
}

function npmNoticeFor(name: string, spec: string): NoticeEntry {
  const dir = npmPackageDir(name)
  const found = dir ? dirTexts(dir) : null
  let licence = found?.licence ?? ''
  let url = `https://www.npmjs.com/package/${encodeURIComponent(name)}`
  if (dir) {
    try {
      const meta = readJSON(join(dir, 'package.json'))
      licence = licence === 'see text' || licence === '' ? npmLicence(meta) : licence
      url = npmHomepage(name, meta)
    } catch {
      licence = licence || 'unknown'
    }
  }
  licence = licence || 'unknown'
  return {
    name: spec.includes('@') ? spec : `${name}@${spec}`,
    licence,
    url,
    texts: found?.texts ?? [`(no LICENSE or NOTICE in ${name}; declared licence: ${licence})\n`],
  }
}

function npmNotices(): NoticeEntry[] {
  const pkgs = parseBunLockPackages(readFileSync(join(repoRoot, 'bun.lock'), 'utf8'))
  const seen = new Set<string>()
  const out: NoticeEntry[] = []
  for (const { name, spec } of pkgs) {
    if (!seen.has(name)) {
      seen.add(name)
      out.push(npmNoticeFor(name, spec))
    }
  }
  return out
}

function collectNotices(): NoticeEntry[] {
  const entries = [...goModuleNotices(), ...npmNotices()]
  entries.sort((a, b) => a.name.localeCompare(b.name))
  return entries
}

function formatNotices(entries: NoticeEntry[]): string {
  const blocks = entries.map((e) => {
    const header = `${e.name}\nlicence: ${e.licence}\n${e.url}`
    return `${header}\n\n${e.texts.join('\n\n').trim()}\n`
  })
  return [
    'Third-party notices',
    '',
    'Go modules compiled into Mortar and npm packages named in bun.lock, with the',
    'licence and NOTICE text from each package directory.',
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
  const deps = asRecord(pkg.dependencies)
  const npm = Object.keys(deps)
    .sort()
    .map((name) => {
      const meta = readJSON(join(frontendRoot, 'node_modules', ...name.split('/'), 'package.json'))
      return { name, licence: npmLicence(meta), url: npmHomepage(name, meta) }
    })

  const goModText = readFileSync(join(repoRoot, 'go.mod'), 'utf8')
  const go = firstRequireBlock(goModText).map((mod) => ({
    name: mod,
    licence: classifyLicenceText(readLicenceFile(goListDir(mod))),
    url: goModuleURL(mod),
  }))

  const extra: CreditEntry[] = [
    {
      name: 'Mortar',
      licence: classifyLicenceText(readLicenceFile(repoRoot)),
      url: 'https://github.com/Rethunk-AI/mortar',
    },
    {
      name: 'Mortar SMAPI Bridge',
      licence: 'AGPL-3.0',
      url: 'https://github.com/Rethunk-AI/mortar-smapi-bridge',
    },
    {
      name: 'SMAPI',
      licence: 'LGPL-3.0',
      url: 'https://smapi.io/',
    },
    {
      name: 'Minigalaxy icon',
      licence: 'GPL-3.0',
      url: 'https://github.com/sharkwouter/minigalaxy',
    },
    {
      name: 'Fedora 44 default wallpaper (f44-01-night)',
      licence: 'CC-BY-SA-4.0',
      url: 'https://fedoraproject.org/wiki/F44_Artwork',
    },
  ]

  const excluded = new Set(['Mortar', 'Mortar SMAPI Bridge', 'SMAPI', 'BepInEx'])
  return [...extra, ...npm, ...go].filter((entry) => !excluded.has(entry.name))
}

function main(): void {
  const entries = buildCredits()
  if (!isCreditList(entries)) {
    throw new Error('generated credits failed shape check')
  }
  const outDir = join(frontendRoot, 'src', 'settings', 'generated')
  mkdirSync(outDir, { recursive: true })
  writeFileSync(join(outDir, 'credits.json'), `${JSON.stringify(entries, null, 2)}\n`)
  writeFileSync(join(repoRoot, 'THIRD_PARTY_NOTICES'), formatNotices(collectNotices()))
}

if (import.meta.main) {
  main()
}

export { classifyLicenceText, collectNotices, parseBunLockPackages }
