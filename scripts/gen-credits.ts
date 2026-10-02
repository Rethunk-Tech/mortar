#!/usr/bin/env bun
import { mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const LICENCE_ERROR_CHARS = 280

const here = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(here, '..')
const frontendRoot = join(repoRoot, 'frontend')
const licenceFileName = /^(license|licence|copying)(\..+)?$/i

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
}

if (import.meta.main) {
  main()
}

export { classifyLicenceText }
