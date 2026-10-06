import { afterAll, describe, expect, test } from 'bun:test'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { classifyLicenceText, collectNotices } from './gen-credits.ts'

describe('classifyLicenceText', () => {
  test('finds licence text after a copyright header', () => {
    const text = `${'Copyright IBM Corp. 2014, 2026\n'.repeat(300)}
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files.`

    expect(classifyLicenceText(text)).toBe('MIT')
  })

  test('recognises both wordings of ISC', () => {
    expect(
      classifyLicenceText('Permission to use, copy, modify, and distribute this software'),
    ).toBe('ISC')
    expect(
      classifyLicenceText('Permission to use, copy, modify, and/or distribute this software'),
    ).toBe('ISC')
  })
})

describe('collectNotices', () => {
  const mitText =
    'MIT License\n\nPermission is hereby granted, free of charge, to any person obtaining a copy'
  const bsdText =
    'Redistribution and use in source and binary forms, with or without modification, are permitted.\nNeither the name'

  const roots: string[] = []
  afterAll(() => {
    for (const root of roots) {
      rmSync(root, { recursive: true, force: true })
    }
  })

  function tree(files: Record<string, string>): string {
    const root = mkdtempSync(join(tmpdir(), 'gen-credits-'))
    roots.push(root)
    for (const [rel, text] of Object.entries(files)) {
      mkdirSync(dirname(join(root, rel)), { recursive: true })
      writeFileSync(join(root, rel), text)
    }
    return root
  }

  test('covers the linked Go modules and bundled packages once each, with their licence and NOTICE text', () => {
    const root = tree({
      'goroot/LICENSE': bsdText,
      'mod/dns/LICENSE': bsdText,
      'mod/dns/NOTICE': 'dns notice',
      'node_modules/react/package.json': '{"name":"react","version":"19.0.0","license":"MIT"}',
      'node_modules/react/LICENSE': mitText,
      'node_modules/react/index.js': '',
      'node_modules/@mui/system/package.json':
        '{"name":"@mui/system","version":"7.0.0","license":"MIT"}',
      'node_modules/@mui/system/LICENSE': mitText,
      'node_modules/@mui/system/a.js': '',
      'node_modules/@mui/system/b.js': '',
      'node_modules/@mui/system/node_modules/react/package.json':
        '{"name":"react","version":"18.0.0","license":"MIT"}',
      'node_modules/@mui/system/node_modules/react/LICENSE': mitText,
      'node_modules/@mui/system/node_modules/react/index.js': '',
    })
    const dnsLine = `github.com/miekg/dns\tv1.1.0\t${join(root, 'mod/dns')}`
    const notices = collectNotices({
      goroot: join(root, 'goroot'),
      goVersion: 'go1.27',
      goDeps: [dnsLine, dnsLine, 'github.com/Rethunk-Tech/mortar\t\t/repo', ''],
      bundleInputs: [
        join(root, 'node_modules/react/index.js'),
        join(root, 'node_modules/@mui/system/a.js'),
        join(root, 'node_modules/@mui/system/b.js'),
        join(root, 'node_modules/@mui/system/node_modules/react/index.js'),
        join(root, 'src/main.tsx'),
      ],
    })
    expect(notices.map((n) => `${n.name} ${n.licence}`)).toEqual(
      expect.arrayContaining([
        'Go standard library go1.27 BSD-3-Clause',
        'github.com/miekg/dns@v1.1.0 BSD-3-Clause',
        'react@19.0.0 MIT',
        'react@18.0.0 MIT',
        '@mui/system@7.0.0 MIT',
      ]),
    )
    expect(notices.filter((n) => n.name.startsWith('github.com/miekg/dns@'))).toHaveLength(1)
    expect(notices.some((n) => n.name.includes('Rethunk-Tech/mortar'))).toBe(false)
    expect(
      notices.find((n) => n.name.startsWith('github.com/miekg/dns@'))?.texts.join('\n'),
    ).toMatch(/dns notice/)
  })

  test('fails on a shipped package with no licence', () => {
    const root = tree({
      'goroot/LICENSE': bsdText,
      'node_modules/bare/package.json': '{"name":"bare","version":"1.0.0"}',
    })
    expect(() =>
      collectNotices({
        goroot: join(root, 'goroot'),
        goVersion: '',
        goDeps: [],
        bundleInputs: [join(root, 'node_modules/bare/index.js')],
      }),
    ).toThrow(/bare@1.0.0/)
  })
})
