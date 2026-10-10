import { beforeEach, expect, mock, test } from 'bun:test'

const calls = { opened: [] as string[], installed: [] as unknown[][] }

// mock.module outlives this file in the one bun test process, so the mock keeps the module's other exports for the files that import them.
const archivesvc = await import(
  '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts'
)
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts', () => ({
  ...archivesvc,
  ItchPage: async (text: string) => {
    const m = /^https:\/\/([a-z0-9-]+)\.itch\.io\/([\w-]+)$/.exec(text)
    if (!m) {
      throw new Error('not an itch.io page')
    }
    return { id: `${m[1]}/${m[2]}`, url: `https://${m[1]}.itch.io/${m[2]}` }
  },
  InstallItchDownload: async (...args: unknown[]) => {
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
const { itchDownloadInstaller, liveItchPage, ITCH_PAGE_TTL_MS, startItchPage, useItchPage } =
  await import('./itch.ts')

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
  useItchPage.getState().set(null)
  useHandoffAsk.setState({ queue: [] })
})

test('a pasted itch.io page opens in the browser and is remembered; anything else is left to the share preview', async () => {
  expect(await startItchPage('https://someone.itch.io/cool-mod')).toBe(true)
  expect(calls.opened).toEqual(['https://someone.itch.io/cool-mod'])
  expect(useItchPage.getState().page).toMatchObject({
    id: 'someone/cool-mod',
    url: 'https://someone.itch.io/cool-mod',
  })

  calls.opened.length = 0
  expect(await startItchPage('https://mortar.rethunk.tech/stardew/p#abc')).toBe(false)
  expect(calls.opened).toEqual([])
  expect(useItchPage.getState().page).toBeNull()
})

const ref = { id: 'someone/cool-mod', url: 'https://someone.itch.io/cool-mod', openedAt: 1_000_000 }
const plain: string[] = []
const install = async (_g: string, _p: string, file: string) => {
  plain.push(file)
  return { profile: {}, added: [] } as never
}

test('the next Downloads install is recorded under the remembered page, once', async () => {
  plain.length = 0
  await itchDownloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'a.zip', mtime: 2_000_000 },
    install,
    2_000_000,
  )()
  expect(plain).toEqual(['a.zip'])
  expect(calls.installed).toEqual([])

  useItchPage.getState().set(ref)
  const run = itchDownloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'b.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
  answer(true)
  await run
  expect(calls.installed).toEqual([['stardew', 'p1', 'b.zip', 'someone/cool-mod']])
  expect(plain).toEqual(['a.zip'])
  expect(useItchPage.getState().page).toBeNull()
})

test('a file saved before the page was opened is not attributed to it', async () => {
  plain.length = 0
  useItchPage.getState().set(ref)
  await itchDownloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'old.zip', mtime: 999_999 },
    install,
    1_500_000,
  )()
  expect(plain).toEqual(['old.zip'])
  expect(calls.installed).toEqual([])
  expect(useItchPage.getState().page).toEqual(ref)
})

test('a page expires after the time-to-live and is forgotten', async () => {
  plain.length = 0
  useItchPage.getState().set(ref)
  const late = ref.openedAt + ITCH_PAGE_TTL_MS + 1
  await itchDownloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'late.zip', mtime: late },
    install,
    late,
  )()
  expect(plain).toEqual(['late.zip'])
  expect(calls.installed).toEqual([])
  expect(useItchPage.getState().page).toBeNull()
})

test('a pasted non-itch.io link forgets an earlier remembered page', async () => {
  useItchPage.getState().set(ref)
  expect(await startItchPage('https://mortar.rethunk.tech/stardew/p#abc')).toBe(false)
  expect(useItchPage.getState().page).toBeNull()
})

test('a page stops being live once it has expired', () => {
  expect(liveItchPage(ref, ref.openedAt + ITCH_PAGE_TTL_MS)).toEqual(ref)
  expect(liveItchPage(ref, ref.openedAt + ITCH_PAGE_TTL_MS + 1)).toBeNull()
  expect(liveItchPage(null)).toBeNull()
})

test('the question names the file and the page and No installs the file as it is, keeping the page for the next file', async () => {
  useItchPage.getState().set(ref)
  plain.length = 0
  const run = itchDownloadInstaller(
    { game: 'stardew', profileId: 'p1', file: '/home/me/Downloads/m.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
  const [ask] = useHandoffAsk.getState().queue
  expect(ask?.file).toBe('m.zip')
  expect(ask?.page).toContain('someone/cool-mod')
  answer(false)
  await run
  expect(plain).toEqual(['/home/me/Downloads/m.zip'])
  expect(calls.installed).toEqual([])
  expect(useItchPage.getState().page).toEqual(ref)
})

test('after Yes the page is no longer waiting for a file, so a second file is not asked', async () => {
  useItchPage.getState().set(ref)
  plain.length = 0
  const first = itchDownloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'one.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
  answer(true)
  await first
  await itchDownloadInstaller(
    { game: 'stardew', profileId: 'p1', file: 'two.zip', mtime: 1_500_000 },
    install,
    1_500_000,
  )()
  expect(useHandoffAsk.getState().queue).toEqual([])
  expect(calls.installed.map((c) => c[2])).toEqual(['one.zip'])
  expect(plain).toEqual(['two.zip'])
})
