import { expect, test } from 'bun:test'
import { nxmShowWithIcon } from './minimisedNotice.ts'

test('a notification includes the installed logo', () => {
  expect(nxmShowWithIcon(7, 'Title', 'Body', '/tmp/mortar-icon.png')).toEqual({
    id: 'nxm-7',
    title: 'Title',
    body: 'Body',
    categoryId: 'nxm-show',
    attachments: [{ id: 'icon', path: '/tmp/mortar-icon.png', type: 'appLogoOverride' }],
  })
})

test('a notification omits the logo when it is not installed', () => {
  expect(nxmShowWithIcon(7, 'Title', 'Body', '')).toEqual({
    id: 'nxm-7',
    title: 'Title',
    body: 'Body',
    categoryId: 'nxm-show',
  })
})
