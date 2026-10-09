import { beforeEach, expect, mock, test } from 'bun:test'

const calls = { opened: [] as string[], installed: [] as unknown[][] }

mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts', () => ({
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

const {
  downloadInstaller,
  livePatreonPost,
  PATREON_POST_TTL_MS,
  startPatreonPost,
  usePatreonPost,
} = await import('./patreon.ts')

beforeEach(() => {
  calls.opened.length = 0
  calls.installed.length = 0
  usePatreonPost.getState().set(null)
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
  await downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'b.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
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
