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

const { downloadInstaller, startPatreonPost, usePatreonPost } = await import('./patreon.ts')

beforeEach(() => {
  calls.opened.length = 0
  calls.installed.length = 0
  usePatreonPost.getState().set(null)
})

test('a pasted Patreon post opens in the browser and is remembered; anything else is left to the share preview', async () => {
  expect(await startPatreonPost('https://www.patreon.com/posts/cool-mod-4242')).toBe(true)
  expect(calls.opened).toEqual(['https://www.patreon.com/posts/4242'])
  expect(usePatreonPost.getState().post).toEqual({
    id: '4242',
    url: 'https://www.patreon.com/posts/4242',
  })

  calls.opened.length = 0
  expect(await startPatreonPost('https://mortar.rethunk.tech/stardew/p#abc')).toBe(false)
  expect(calls.opened).toEqual([])
  expect(usePatreonPost.getState().post).toBeNull()
})

test('the next Downloads install is recorded under the remembered post, once', async () => {
  const plain: string[] = []
  const install = async (_g: string, _p: string, file: string) => {
    plain.push(file)
    return { profile: {}, added: [] } as never
  }
  await downloadInstaller('stardew', 'p1', 'a.zip', install)()
  expect(plain).toEqual(['a.zip'])
  expect(calls.installed).toEqual([])

  usePatreonPost.getState().set({ id: '4242', url: 'https://www.patreon.com/posts/4242' })
  await downloadInstaller('stardew', 'p1', 'b.zip', install)()
  expect(calls.installed).toEqual([['stardew', 'p1', 'b.zip', '4242']])
  expect(plain).toEqual(['a.zip'])
  expect(usePatreonPost.getState().post).toBeNull()
})
