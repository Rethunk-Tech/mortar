import { describe, expect, test } from 'bun:test'
import { overlayFileUrl } from './overlayUrl.ts'

describe('overlayFileUrl', () => {
  test('sets port and token query names the overlay page reads', () => {
    const got = overlayFileUrl('/home/u/.local/share/mortar/overlay/index.html', 8123, 'ab&c')
    const url = new URL(got)
    expect(url.protocol).toBe('file:')
    expect(url.pathname).toBe('/home/u/.local/share/mortar/overlay/index.html')
    expect(url.searchParams.get('port')).toBe('8123')
    expect(url.searchParams.get('token')).toBe('ab&c')
  })
})
