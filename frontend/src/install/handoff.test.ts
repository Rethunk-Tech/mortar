import { beforeEach, expect, mock, test } from 'bun:test'

const calls = { opened: [] as string[], installed: [] as unknown[][] }

// mock.module outlives this file in the one bun test process, so the mock keeps the module's other exports for the files that import them.
const archivesvc = await import(
  '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts'
)
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts', () => ({
  ...archivesvc,
  HandoffPage: async (text: string) => {
    const post = /^https:\/\/www\.patreon\.com\/posts\/[\w-]*?-?(\d+)$/.exec(text)
    if (post) {
      return { kind: 'patreon', id: post[1], url: `https://www.patreon.com/posts/${post[1]}` }
    }
    const page = /^https:\/\/([a-z0-9-]+)\.itch\.io\/([\w-]+)$/.exec(text)
    if (page) {
      return {
        kind: 'itch',
        id: `${page[1]}/${page[2]}`,
        url: `https://${page[1]}.itch.io/${page[2]}`,
      }
    }
    throw new Error('not a handoff page')
  },
  InstallHandoffDownload: async (...args: unknown[]) => {
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
const { downloadInstaller, liveHandoff, HANDOFF_TTL_MS, startHandoff, useHandoff } = await import(
  './handoff.ts'
)

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
  useHandoff.getState().set(null)
  useHandoffAsk.setState({ queue: [] })
})

test('a pasted Patreon post opens in the browser and is remembered; anything else is left to the share preview', async () => {
  expect(await startHandoff('https://www.patreon.com/posts/cool-mod-4242')).toBe(true)
  expect(calls.opened).toEqual(['https://www.patreon.com/posts/4242'])
  expect(useHandoff.getState().page).toMatchObject({
    kind: 'patreon',
    id: '4242',
    url: 'https://www.patreon.com/posts/4242',
  })

  calls.opened.length = 0
  expect(await startHandoff('https://mortar.rethunk.tech/stardew/p#abc')).toBe(false)
  expect(calls.opened).toEqual([])
  expect(useHandoff.getState().page).toBeNull()
})

const ref = {
  kind: 'patreon',
  id: '4242',
  url: 'https://www.patreon.com/posts/4242',
  openedAt: 1_000_000,
}
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

  useHandoff.getState().set(ref)
  const run = downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'b.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
  answer(true)
  await run
  expect(calls.installed).toEqual([['stardew', 'p1', 'b.zip', 'patreon', '4242']])
  expect(plain).toEqual(['a.zip'])
  expect(useHandoff.getState().page).toBeNull()
})

test('a file saved before the post was opened is not attributed to it', async () => {
  plain.length = 0
  useHandoff.getState().set(ref)
  await downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'old.zip', mtime: 999_999 },
    install,
    1_500_000,
  )()
  expect(plain).toEqual(['old.zip'])
  expect(calls.installed).toEqual([])
  expect(useHandoff.getState().page).toEqual(ref)
})

test('a post expires after the time-to-live and is forgotten', async () => {
  plain.length = 0
  useHandoff.getState().set(ref)
  const late = ref.openedAt + HANDOFF_TTL_MS + 1
  await downloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'late.zip', mtime: late },
    install,
    late,
  )()
  expect(plain).toEqual(['late.zip'])
  expect(calls.installed).toEqual([])
  expect(useHandoff.getState().page).toBeNull()
})

test('an itch.io page replaces a remembered Patreon post, and its file is recorded under the page by its plain name', async () => {
  useHandoff.getState().set(ref)
  expect(await startHandoff('https://someone.itch.io/cool-mod')).toBe(true)
  expect(calls.opened).toEqual(['https://someone.itch.io/cool-mod'])
  const opened = useHandoff.getState().page
  expect(opened).toMatchObject({ kind: 'itch', id: 'someone/cool-mod' })
  const at = (opened?.openedAt ?? 0) + 1
  const run = downloadInstaller(
    { game: 'sims4', profileId: 'p1', file: 'c.zip', mtime: at },
    install,
    at,
  )()
  expect(useHandoffAsk.getState().queue[0]?.page).toBe('someone/cool-mod')
  answer(true)
  await run
  expect(calls.installed).toEqual([['sims4', 'p1', 'c.zip', 'itch', 'someone/cool-mod']])
})

test('a pasted non-Patreon link forgets an earlier remembered post', async () => {
  useHandoff.getState().set(ref)
  expect(await startHandoff('https://mortar.rethunk.tech/stardew/p#abc')).toBe(false)
  expect(useHandoff.getState().page).toBeNull()
})

test('a post stops being live once it has expired', () => {
  expect(liveHandoff(ref, ref.openedAt + HANDOFF_TTL_MS)).toEqual(ref)
  expect(liveHandoff(ref, ref.openedAt + HANDOFF_TTL_MS + 1)).toBeNull()
  expect(liveHandoff(null)).toBeNull()
})

test('the question names the file and the post and No installs the file as it is, keeping the post for the next file', async () => {
  useHandoff.getState().set(ref)
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
  expect(useHandoff.getState().page).toEqual(ref)
})

test('after Yes the post is no longer waiting for a file, so a second file is not asked', async () => {
  useHandoff.getState().set(ref)
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
