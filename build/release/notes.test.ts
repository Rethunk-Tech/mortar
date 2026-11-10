import { afterAll, beforeAll, describe, expect, test } from 'bun:test'
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const script = join(import.meta.dir, 'notes.sh')
let repo = ''

// Git's own variables are dropped: run from a hook or a worktree, GIT_DIR would point every command, `git config`
// included, at the repository running the test instead of the throwaway one.
const outside = Object.fromEntries(
  Object.entries(process.env).filter(([k]) => !k.startsWith('GIT_')),
)

// Merged after the caller's env, so neither the machine's global and system git config nor a stray variable can put
// signing, hooks or fsmonitor back on the throwaway repo.
const isolated = {
  GIT_CONFIG_GLOBAL: '/dev/null',
  GIT_CONFIG_NOSYSTEM: '1',
}

function run(cmd: string[], env: Record<string, string> = {}, cwd = repo, input?: string): string {
  const p = Bun.spawnSync(cmd, {
    cwd,
    env: { ...outside, ...env, ...isolated },
    ...(input === undefined ? {} : { stdin: Buffer.from(input) }),
  })
  if (p.exitCode !== 0) {
    throw new Error(`${cmd.join(' ')}: ${p.stderr.toString()}`)
  }
  return p.stdout.toString()
}

function git(dir: string, ...args: string[]): string {
  return run(
    [
      'git',
      '-c',
      'core.hooksPath=/dev/null',
      '-c',
      'core.fsmonitor=false',
      '-c',
      'commit.gpgsign=false',
      '-c',
      'tag.gpgSign=false',
      '-c',
      'gc.auto=0',
      ...args,
    ],
    {},
    dir,
  )
}

// The commits the release notes read, oldest first, and the tag each one carries.
const commits: [string, tag?: string][] = [
  ['feat(mods): before the previous release', 'v0.1.0'],
  ['feat(mods): search matches category names', 'components'],
  ['fix: a crash on start.'],
  ['perf(profiles): faster profile switch'],
  ['feat(launchsvc): internal service change'],
  ['fix(ui): RemoveDriftFolder returns an undo token'],
  ['chore(deps): bump everything'],
  ['docs: explain the thing'],
  ['refactor(mods): move code'],
  ['feat(regress): the matrix measures launch (a)'],
  ['feat(launch): a server build honours MORTAR_LAUNCH_WRAPPER'],
  ['feat(saves)!: back up <saves> & restore'],
  ['feat(games): 104 more BepInEx 5 games from Thunderstore'],
]

const ident = 'notes test <notes@test.invalid>'

// fast-import writes the whole history and its tags in one process: a `git commit` per entry costs a repo open, an
// index read and a ref update each, and under gate load the fifteen processes together outran bun's hook budget.
// Its stream command set also bypasses hooks, signing and the index, so the host's git setup cannot reach it.
function importStream(): string {
  const out: string[] = []
  commits.forEach(([subject, tag], i) => {
    out.push(
      'commit refs/heads/main',
      `mark :${i + 1}`,
      `committer ${ident} ${1_700_000_000 + i} +0000`,
      // The subject is the whole commit message, so the newline `join` adds below ends the data block.
      `data ${Buffer.byteLength(subject)}\n${subject}`,
    )
    if (i > 0) {
      out.push(`from :${i}`)
    }
    if (tag) {
      out.push(`reset refs/tags/${tag}`, `from :${i + 1}`)
    }
  })
  return `${out.join('\n')}\n`
}

function build(dir: string): void {
  git(dir, 'init', '-q', '-b', 'main')
  run(['git', 'fast-import', '--quiet', '--done'], {}, dir, `${importStream()}done\n`)
}

beforeAll(() => {
  repo = mkdtempSync(join(tmpdir(), 'mortar-notes-'))
  build(repo)
})

afterAll(() => rmSync(repo, { recursive: true, force: true }))

describe('notes.sh', () => {
  test('builds its fixture over a repo whose hooks and signing are broken', () => {
    const dir = mkdtempSync(join(tmpdir(), 'mortar-notes-hostile-'))
    const hooks = mkdtempSync(join(tmpdir(), 'mortar-notes-hooks-'))
    writeFileSync(join(hooks, 'pre-commit'), '#!/bin/sh\nexit 1\n', { mode: 0o755 })
    try {
      git(dir, 'init', '-q', '-b', 'main')
      git(dir, 'config', 'core.hooksPath', hooks)
      git(dir, 'config', 'commit.gpgsign', 'true')
      git(dir, 'config', 'tag.gpgSign', 'true')
      git(dir, 'config', 'core.fsmonitor', 'false')
      build(dir)
      // The whole history and both tags landed, so the hostile config did not silently drop a commit.
      expect(git(dir, 'rev-list', '--count', 'HEAD')).toBe(`${commits.length}\n`)
      expect(git(dir, 'tag', '--list')).toBe('components\nv0.1.0\n')
    } finally {
      rmSync(dir, { recursive: true, force: true })
      rmSync(hooks, { recursive: true, force: true })
    }
  })

  test('keeps user-facing feat, fix and perf since the previous v* tag, in sentence case under their areas', () => {
    const out = run([script, 'v0.2.0', 'HEAD'], { GITHUB_REPOSITORY: 'o/r' })
    expect(out).toBe(
      [
        '## New',
        '',
        '### Games',
        '',
        '- 104 more BepInEx 5 games from Thunderstore',
        '',
        '### Mods and browsing',
        '',
        '- Search matches category names',
        '',
        '### Saves',
        '',
        '- Back up <saves> & restore',
        '',
        '## Fixed',
        '',
        '### Profiles and sharing',
        '',
        '- Faster profile switch',
        '',
        '### Other',
        '',
        '- A crash on start',
        '',
        'The Linux AppImage and portable program need glibc 2.35 or newer (Ubuntu 22.04, Linux Mint 21, Debian 12, Fedora 36 or later); older systems use the Flatpak.',
        '',
        '[Full changelog](https://github.com/o/r/compare/v0.1.0...v0.2.0)',
        '',
      ].join('\n'),
    )
  })

  test('caps each group and counts the rest', () => {
    const out = run([script, 'v0.2.0', 'HEAD'], { NOTES_CAP: '1' })
    expect(out).toContain(
      '### Games\n\n- 104 more BepInEx 5 games from Thunderstore\n- …and 2 more\n',
    )
    expect(out).not.toContain('### Mods and browsing')
  })

  test('a release notes file replaces the commit subjects, uncapped, with a Changed group', () => {
    const file = join(repo, 'v0.2.0.md')
    writeFileSync(
      file,
      '## New\n\n- Lethal Company gets a Load order tab\n- CurseForge is a source\n\n## Changed\n\n- Settings moved\n\n## Fixed\n\n- No more crash on start\n',
    )
    const out = run([script, 'v0.2.0', 'HEAD'], { NOTES_FILE: file, NOTES_CAP: '1' })
    expect(out).toContain(
      '## New\n\n- Lethal Company gets a Load order tab\n- CurseForge is a source\n\n## Changed\n\n- Settings moved\n\n## Fixed\n\n- No more crash on start\n',
    )
    expect(out).not.toContain('Back up')
  })

  test('area headings in a release notes file group the release page and stay out of the AppStream description', () => {
    const file = join(repo, 'v0.2.0.md')
    writeFileSync(
      file,
      '## Fixed\n\n### Saves\n\n- Slots list once\n\n### Games\n\n- Profiles start\n',
    )
    const out = run([script, 'v0.2.0', 'HEAD'], { NOTES_FILE: file })
    expect(out).toContain(
      '## Fixed\n\n### Saves\n\n- Slots list once\n\n### Games\n\n- Profiles start\n',
    )
  })

  test('writes an escaped AppStream description into the release element', () => {
    const file = join(repo, 'metainfo.xml')
    writeFileSync(
      file,
      '<releases>\n    <release version="0.2.0" date="2026-10-04" />\n    <release version="0.1.0" date="2026-01-01" />\n</releases>\n',
    )
    // Both clocks in UTC: bun test runs in UTC while the script's `date +%F` would read the local zone.
    run([script, '--metainfo', file, 'v0.2.0', 'HEAD'], { TZ: 'UTC' })
    run([script, '--metainfo', file, 'v0.2.0', 'HEAD'], { TZ: 'UTC' })
    const xml = readFileSync(file, 'utf8')
    // The release being written is dated today; others keep theirs.
    const today = new Date().toISOString().slice(0, 10)
    expect(xml).toContain(`<release version="0.2.0" date="${today}">\n      <description>`)
    expect(xml).toContain('<li>Back up &lt;saves&gt; &amp; restore</li>')
    expect(xml.match(/<description>/g)?.length).toBe(1)
    expect(xml).toContain('    </release>\n    <release version="0.1.0" date="2026-01-01" />')
  })
})
