import { afterAll, beforeAll, describe, expect, test } from 'bun:test'
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const script = join(import.meta.dir, 'notes.sh')
let repo = ''

function run(cmd: string[], env: Record<string, string> = {}): string {
  const p = Bun.spawnSync(cmd, { cwd: repo, env: { ...process.env, ...env } })
  if (p.exitCode !== 0) {
    throw new Error(`${cmd.join(' ')}: ${p.stderr.toString()}`)
  }
  return p.stdout.toString()
}

function commit(subject: string): void {
  run(['git', 'commit', '-q', '--allow-empty', '-m', subject])
}

beforeAll(() => {
  repo = mkdtempSync(join(tmpdir(), 'mortar-notes-'))
  run(['git', 'init', '-q'])
  run(['git', 'config', 'user.email', 'notes@test.invalid'])
  run(['git', 'config', 'user.name', 'notes test'])
  commit('feat(mods): before the previous release')
  run(['git', 'tag', 'v0.1.0'])
  commit('feat(mods): search matches category names')
  run(['git', 'tag', 'components'])
  commit('fix: a crash on start.')
  commit('perf(profiles): faster profile switch')
  commit('feat(launchsvc): internal service change')
  commit('fix(ui): RemoveDriftFolder returns an undo token')
  commit('chore(deps): bump everything')
  commit('docs: explain the thing')
  commit('refactor(mods): move code')
  commit('feat(regress): the matrix measures launch (a)')
  commit('feat(launch): a server build honours MORTAR_LAUNCH_WRAPPER')
  commit('feat(saves)!: back up <saves> & restore')
})

afterAll(() => rmSync(repo, { recursive: true, force: true }))

describe('notes.sh', () => {
  test('keeps user-facing feat, fix and perf since the previous v* tag, in sentence case', () => {
    const out = run([script, 'v0.2.0', 'HEAD'], { GITHUB_REPOSITORY: 'o/r' })
    expect(out).toBe(
      [
        '## New',
        '',
        '- Back up <saves> & restore',
        '- Search matches category names',
        '',
        '## Fixed',
        '',
        '- Faster profile switch',
        '- A crash on start',
        '',
        'The Linux AppImage and portable program need glibc 2.38 or newer (Ubuntu 24.04, Debian 13, Fedora 39 or later); older systems use the Flatpak.',
        '',
        '[Full changelog](https://github.com/o/r/compare/v0.1.0...v0.2.0)',
        '',
      ].join('\n'),
    )
  })

  test('caps each group and counts the rest', () => {
    const out = run([script, 'v0.2.0', 'HEAD'], { NOTES_CAP: '1' })
    expect(out).toContain('- Back up <saves> & restore\n- …and 1 more\n')
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

  test('writes an escaped AppStream description into the release element', () => {
    const file = join(repo, 'metainfo.xml')
    writeFileSync(
      file,
      '<releases>\n    <release version="0.2.0" date="2026-10-04" />\n    <release version="0.1.0" date="2026-01-01" />\n</releases>\n',
    )
    run([script, '--metainfo', file, 'v0.2.0', 'HEAD'])
    run([script, '--metainfo', file, 'v0.2.0', 'HEAD'])
    const xml = readFileSync(file, 'utf8')
    expect(xml).toContain('<release version="0.2.0" date="2026-10-04">\n      <description>')
    expect(xml).toContain('<li>Back up &lt;saves&gt; &amp; restore</li>')
    expect(xml.match(/<description>/g)?.length).toBe(1)
    expect(xml).toContain('    </release>\n    <release version="0.1.0" date="2026-01-01" />')
  })
})
