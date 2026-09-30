import { describe, expect, test } from 'bun:test'
import { overlayPageUrl, overlayPreview } from './overlayUrl.ts'

const base = 'file:///tmp/overlay/index.html?port=18765&token=tok'

describe('overlayPageUrl', () => {
  test('all values omits field', () => {
    expect(overlayPageUrl(base)).toBe(base)
  })

  test('per-field query', () => {
    expect(overlayPageUrl(base, 'money')).toBe(`${base}&field=money`)
    expect(overlayPageUrl(base, 'skill.mining')).toBe(`${base}&field=skill.mining`)
  })

  test('label flag', () => {
    expect(overlayPageUrl(base, undefined, true)).toBe(`${base}&label=1`)
    expect(overlayPageUrl(base, 'time', true)).toBe(`${base}&field=time&label=1`)
  })
})

describe('overlayPreview', () => {
  test('unreachable', () => {
    expect(overlayPreview(null, 'money')).toEqual({ kind: 'unreachable' })
    expect(overlayPreview({ ok: false }, 'money')).toEqual({ kind: 'unreachable' })
  })

  test('not in game', () => {
    expect(overlayPreview({ ok: true, body: { inGame: false } }, 'money')).toEqual({
      kind: 'notInGame',
    })
  })

  test('value from snapshot', () => {
    const body = {
      inGame: true,
      money: 1234,
      timeOfDay: 1610,
      skills: { mining: 6 },
    }
    expect(overlayPreview({ ok: true, body }, 'money')).toEqual({ kind: 'value', text: '1234g' })
    expect(overlayPreview({ ok: true, body }, 'time')).toEqual({ kind: 'value', text: '4:10 PM' })
    expect(overlayPreview({ ok: true, body }, 'skill.mining')).toEqual({ kind: 'value', text: '6' })
  })
})
