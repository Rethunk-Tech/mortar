import { expect, test } from 'bun:test'
import { anchorProblems, anchorsOf, reposOf } from './site-downloads.ts'

const latest = {
  'Rethunk-Tech/mortar': {
    tag_name: 'v1.2.3',
    assets: [{ name: 'mortar_1.2.3_amd64.deb' }, { name: 'mortar-amd64-installer.exe' }],
  },
}

test('a link resolves when its asset, with the version filled in, is on the latest release', () => {
  const anchors = anchorsOf(
    'p.html',
    `<a data-asset="mortar_VERSION_amd64.deb" href="https://github.com/Rethunk-Tech/mortar/releases/latest">.deb</a>
     <a data-asset="mortar-amd64-installer.exe" href="https://github.com/Rethunk-Tech/mortar/releases/latest/download/mortar-amd64-installer.exe">x</a>`,
  )
  expect(reposOf(anchors)).toEqual(['Rethunk-Tech/mortar'])
  expect(anchorProblems(anchors, latest)).toEqual([])
})

test('a missing asset and an href that disagrees with its data-asset both fail', () => {
  const anchors = anchorsOf(
    'p.html',
    `<a href="https://github.com/Rethunk-Tech/mortar/releases/latest/download/mortar-x.AppImage">a</a>
     <a data-asset="mortar-VERSION-1.x86_64.rpm" href="https://github.com/Rethunk-Tech/mortar/releases/latest">r</a>
     <a data-asset="mortar_VERSION_amd64.deb" href="https://github.com/Rethunk-Tech/mortar/releases/latest/download/mortar-amd64-installer.exe">d</a>`,
  )
  expect(anchorProblems(anchors, latest)).toEqual([
    'p.html: https://github.com/Rethunk-Tech/mortar/releases/latest/download/mortar-x.AppImage names no asset of Rethunk-Tech/mortar v1.2.3',
    'p.html: data-asset mortar-VERSION-1.x86_64.rpm is mortar-1.2.3-1.x86_64.rpm, which Rethunk-Tech/mortar v1.2.3 lacks',
    'p.html: href asset mortar-amd64-installer.exe differs from data-asset mortar_1.2.3_amd64.deb',
  ])
})
