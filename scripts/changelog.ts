// Release notes grouped by what the user sees: the feat, fix and perf commits since the last v* tag, one section per
// area, as Markdown on stdout. Usage: bun scripts/changelog.ts [REF]   (REF defaults to HEAD)
// build/release/notes.sh writes the release page's New and Fixed lists; this groups the same commits by area.

const AREAS: Record<string, string[]> = {
  'Mods and browsing': [
    'browse',
    'mods',
    'mod',
    'source',
    'catalog',
    'templates',
    'bundles',
    'thunderstore',
    'modrinth',
    'nexus',
  ],
  'Profiles and sharing': [
    'profile',
    'profiles',
    'share',
    'packs',
    'pack',
    'import',
    'history',
    'saves',
    'sync',
    'lan',
  ],
  'Launching and loaders': [
    'launch',
    'loader',
    'smapi',
    'bepinex',
    'bepinex5',
    'game',
    'games',
    'steam',
    'deploy',
  ],
  'Problems and diagnostics': ['problems', 'diag', 'console', 'doctor', 'support'],
  'Downloads and updates': ['queue', 'updates', 'watch', 'nxm'],
  'Interface and settings': [
    'settings',
    'ui',
    'ux',
    'theme',
    'shell',
    'frontend',
    'firstrun',
    'tour',
    'i18n',
    'a11y',
    'config',
  ],
}
const OTHER = 'Other'

// Scopes that name code or tooling, and subjects that name code, never reach a user (as in build/release/notes.sh).
const INTERNAL =
  /^([a-z-]*svc|main|release|gate|lint|deps|build|ci|tests?|selftest|dev|fsx|datadir|data|meta|usererr|github|site|sampler|notices|credits|migrate|components|nativehost|errors|logs?|cli|e2e)$/
const CODEISH = /[a-z][A-Z]|[a-z]_[a-z]|`/
const SHOWN = new Set(['feat', 'fix', 'perf'])

const SUBJECT = /^([a-z]+)(?:\(([^)]*)\))?!?: (.+)$/

function areaOf(scope: string): string {
  return Object.keys(AREAS).find((area) => AREAS[area].includes(scope)) ?? OTHER
}

function entry(subject: string): { area: string; line: string } | undefined {
  const m = SUBJECT.exec(subject)
  const [, type = '', scope = '', text = ''] = m ?? []
  const shown = m !== null && SHOWN.has(type) && !INTERNAL.test(scope) && !CODEISH.test(text)
  if (!shown) {
    return undefined
  }
  const line = text.replace(/\.$/, '')
  return {
    area: areaOf(scope),
    line: `- ${type === 'fix' ? 'Fixed: ' : ''}${line[0].toUpperCase()}${line.slice(1)}`,
  }
}

function build(subjects: string[]): string {
  const byArea = new Map<string, string[]>()
  for (const e of subjects.map(entry)) {
    if (e) {
      byArea.set(e.area, [...(byArea.get(e.area) ?? []), e.line])
    }
  }
  const order = [...Object.keys(AREAS), OTHER].filter((a) => byArea.has(a))
  if (order.length === 0) {
    return 'No user-facing changes.\n'
  }
  return order.map((a) => `## ${a}\n\n${byArea.get(a)?.join('\n')}\n`).join('\n')
}

async function git(...args: string[]): Promise<string> {
  const p = Bun.spawn(['git', ...args], { stdout: 'pipe', stderr: 'pipe' })
  const [out, code] = await Promise.all([new Response(p.stdout).text(), p.exited])
  return code === 0 ? out.trim() : ''
}

if (import.meta.main) {
  const ref = process.argv[2] ?? 'HEAD'
  const prev = await git('describe', '--tags', '--match', 'v*', '--abbrev=0', ref)
  const log = await git('log', '--no-merges', '--format=%s', prev ? `${prev}..${ref}` : ref)
  process.stdout.write(build(log.split('\n').filter(Boolean)))
}

export { build }
