import { describe, expect, test } from 'bun:test'
import { overlayFields, overlayPageUrl, overlayPreview, STARDEW_FIELDS } from './overlayUrl.ts'

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

describe('Stardew layout fields', () => {
  const read = (body: Record<string, unknown>, field: string) =>
    overlayPreview({ ok: true, body: { inGame: true, ...body } }, field)

  test('values from the extra snapshot keys', () => {
    const body = {
      dayOfWeek: 'Tuesday',
      daysLeftInSeason: 9,
      friendName: 'Abigail',
      friendHearts: 8,
      petName: 'Rex',
      petType: 'Dog',
      petWatered: false,
    }
    expect(read(body, 'weekday')).toEqual({ kind: 'value', text: 'Tuesday' })
    expect(read(body, 'daysLeft')).toEqual({ kind: 'value', text: '9' })
    expect(read(body, 'friend')).toEqual({ kind: 'value', text: 'Abigail 8' })
    expect(read(body, 'pet')).toEqual({ kind: 'value', text: 'Rex (Dog)' })
    expect(read(body, 'petWatered')).toEqual({ kind: 'value', text: 'No' })
  })

  test('keys the bridge omits read as empty text', () => {
    for (const field of ['weekday', 'daysLeft', 'friend', 'pet', 'petWatered']) {
      expect(read({}, field)).toEqual({ kind: 'value', text: '' })
    }
  })
})

describe('per-game overlay fields', () => {
  test("a game without a layout of its own gets Stardew Valley's", () => {
    expect(overlayFields('stardew')).toBe(STARDEW_FIELDS)
    expect(overlayFields('lethal-company')).toContain('quota')
    expect(overlayFields('valheim')).toContain('bosses')
  })

  test('Lethal Company values', () => {
    const body = {
      inGame: true,
      game: 'lethal-company',
      playerName: 'Zed',
      crewAlive: 2,
      crewTotal: 3,
      quota: 400,
      quotaProgress: 150,
      credits: 90,
    }
    const read = (field: string) => overlayPreview({ ok: true, body }, field)
    expect(read('crew')).toEqual({ kind: 'value', text: '2/3 alive' })
    expect(read('quota')).toEqual({ kind: 'value', text: '150/400' })
    expect(read('credits')).toEqual({ kind: 'value', text: '\u25A090' })
    expect(read('moon')).toEqual({ kind: 'value', text: '' })
  })

  test('Valheim values', () => {
    const body = {
      inGame: true,
      game: 'valheim',
      bossCount: 2,
      bossesDefeated: ['Eikthyr', 'The Elder'],
    }
    expect(overlayPreview({ ok: true, body }, 'bosses')).toEqual({
      kind: 'value',
      text: '2/7 defeated',
    })
    expect(overlayPreview({ ok: true, body }, 'bossList')).toEqual({
      kind: 'value',
      text: 'Eikthyr, The Elder',
    })
  })
})
