import { describe, expect, test } from 'bun:test'

const placeholder = 'frontend/dist/.gitkeep'

// A fresh clone or git worktree has no built UI, and `//go:embed all:frontend/dist` refuses a directory with no
// files, so the Go build, vet and lint fail there unless a tracked file keeps the directory.
describe('frontend/dist embed placeholder', () => {
  test(`${placeholder} is tracked and not ignored`, () => {
    expect(Bun.spawnSync(['git', 'ls-files', '--error-unmatch', placeholder]).exitCode).toBe(0)
    expect(Bun.spawnSync(['git', 'check-ignore', placeholder]).exitCode).toBe(1)
  })

  // The negation that keeps the placeholder must not unignore what the build writes there.
  test('built files stay ignored', () => {
    expect(Bun.spawnSync(['git', 'check-ignore', 'frontend/dist/index.html']).exitCode).toBe(0)
  })
})
