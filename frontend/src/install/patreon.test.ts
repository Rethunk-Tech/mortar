import { beforeEach, expect, mock, test } from 'bun:test'

const calls = { opened: [] as string[], installed: [] as unknown[][] }

// mock.module outlives this file in the one bun test process, so the mock keeps the module's other exports for the files that import them.
const archivesvc = await import(
  '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts'
)
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts', () => ({
  ...archivesvc,
  PatreonPost: async (text: string) => {
    const m = /^https:\/\/www\.patreon\.com\/posts\/[\w-]*?-?(\d+)$/.exec(text)
    if (!m) {
      throw new Error('not a Patreon post')
    }
    return { id: m[1], url: `https://www.patreon.com/posts/${m[1]}` }
  },
  InstallPatreonDownload: async (...args: unknown[]) => {
    calls.installed.push(args)
    return { added: [] }
  },
}))
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts', () => ({
  OpenWeb: async (url: string) => {
    calls.opened.push(url)
  },
}))

const { useHandoffAsk } = await import('./handoffConfirm.ts')
const {
  downloadInstaller,
  livePatreonPost,
  PATREON_POST_TTL_MS,
  startPatreonPost,
  usePatreonPost,
} = await import('./patreon.ts')

// The player's answer to the "Install <file> as from <page>?" question the claiming install is waiting on.
const answer = (yes: boolean) => {
  const { queue, shift } = useHandoffAsk.getState()
  const [ask] = queue
  shift()
  ask?.answer(yes)
}

beforeEach(() => {
  calls.opened.length = 0
  calls.installed.length = 0
  usePatreonPost.getState().set(null)
  useHandoffAsk.setState({ queue: [] })
})

test('a pasted Patreon post opens in the browser and is remembered; anything else is left to the share preview', async () => {
  expect(await startPatreonPost('https://www.patreon.com/posts/cool-mod-4242')).toBe(true)
  expect(calls.opened).toEqual(['https://www.patreon.com/posts/4242'])
  expect(usePatreonPost.getState().post).toMatchObject({
    id: '4242',
    url: 'https://www.patreon.com/posts/4242',
  })

  calls.opened.length = 0
  expect(await startPatreonPost('https://mortar.rethunk.tech/stardew/p#abc')).toBe(false)
  expect(calls.opened).toEqual([])
  expect(usePatreonPost.getState().post).toBeNull()
})

const ref = { id: '4242', url: 'https://www.patreon.com/posts/4242', openedAt: 1_000_000 }
const plain: string[] = []
const install = async (_g: string, _p: string, file: string) => {
  plain.push(file)
  return { profile: {}, added: [] } as never
}

test('the next Downloads install is recorded under the remembered post, once', async () => {
  plain.length = 0
  await downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'a.zip', mtime: 2_000_000 },
    install,
    2_000_000,
  )()
  expect(plain).toEqual(['a.zip'])
  expect(calls.installed).toEqual([])

  usePatreonPost.getState().set(ref)
  const run = downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'b.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
  answer(true)
  await run
  expect(calls.installed).toEqual([['stardew', 'p1', 'b.zip', '4242']])
  expect(plain).toEqual(['a.zip'])
  expect(usePatreonPost.getState().post).toBeNull()
})

test('a file saved before the post was opened is not attributed to it', async () => {
  plain.length = 0
  usePatreonPost.getState().set(ref)
  await downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'old.zip', mtime: 999_999 },
    install,
    1_500_000,
  )()
  expect(plain).toEqual(['old.zip'])
  expect(calls.installed).toEqual([])
  expect(usePatreonPost.getState().post).toEqual(ref)
})

test('a post expires after the time-to-live and is forgotten', async () => {
  plain.length = 0
  usePatreonPost.getState().set(ref)
  const late = ref.openedAt + PATREON_POST_TTL_MS + 1
  await downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'late.zip', mtime: late },
    install,
    late,
  )()
  expect(plain).toEqual(['late.zip'])
  expect(calls.installed).toEqual([])
  expect(usePatreonPost.getState().post).toBeNull()
})

test('a pasted non-Patreon link forgets an earlier remembered post', async () => {
  usePatreonPost.getState().set(ref)
  expect(await startPatreonPost('https://mortar.rethunk.tech/stardew/p#abc')).toBe(false)
  expect(usePatreonPost.getState().post).toBeNull()
})

test('a post stops being live once it has expired', () => {
  expect(livePatreonPost(ref, ref.openedAt + PATREON_POST_TTL_MS)).toEqual(ref)
  expect(livePatreonPost(ref, ref.openedAt + PATREON_POST_TTL_MS + 1)).toBeNull()
  expect(livePatreonPost(null)).toBeNull()
})

test('the question names the file and the post and No installs the file as it is, keeping the post for the next file', async () => {
  usePatreonPost.getState().set(ref)
  plain.length = 0
  const run = downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: '/home/me/Downloads/m.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
  const [ask] = useHandoffAsk.getState().queue
  expect(ask?.file).toBe('m.zip')
  expect(ask?.page).toContain('4242')
  answer(false)
  await run
  expect(plain).toEqual(['/home/me/Downloads/m.zip'])
  expect(calls.installed).toEqual([])
  expect(usePatreonPost.getState().post).toEqual(ref)
})

test('after Yes the post is no longer waiting for a file, so a second file is not asked', async () => {
  usePatreonPost.getState().set(ref)
  plain.length = 0
  const first = downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'one.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
  answer(true)
  await first
  await downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'two.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
  expect(useHandoffAsk.getState().queue).toEqual([])
  expect(calls.installed.map((c) => c[2])).toEqual(['one.zip'])
  expect(plain).toEqual(['two.zip'])
})
