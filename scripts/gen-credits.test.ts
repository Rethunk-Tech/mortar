import { describe, expect, test } from 'bun:test'
import { classifyLicenceText, collectNotices } from './gen-credits.ts'

describe('classifyLicenceText', () => {
  test('finds licence text after a copyright header', () => {
    const text = `${'Copyright IBM Corp. 2014, 2026\n'.repeat(300)}
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files.`

    expect(classifyLicenceText(text)).toBe('MIT')
  })
})

describe('collectNotices', () => {
  test('covers only what ships, with real licence text', async () => {
    const notices = await collectNotices()
    const names = notices.map((n) => n.name)
    const blob = notices.map((n) => n.texts.join('\n')).join('\n')
    expect(blob).not.toMatch(/no LICENSE or NOTICE/)
    expect(names.some((n) => n.startsWith('vite@'))).toBe(false)
    expect(names.some((n) => n.includes('@biomejs/'))).toBe(false)
    expect(names.some((n) => n.startsWith('@babel/core@') || n.startsWith('@jest/'))).toBe(false)
    expect(names.some((n) => n.startsWith('Go standard library'))).toBe(true)
    expect(names.some((n) => n.startsWith('github.com/go-ole/go-ole@'))).toBe(true)
    expect(names.some((n) => n.startsWith('github.com/miekg/dns@'))).toBe(true)
    expect(names.some((n) => n.startsWith('github.com/hashicorp/golang-lru/v2@'))).toBe(true)
    expect(names.some((n) => n.includes('@mui/system@'))).toBe(true)
    expect(names.some((n) => n.includes('@emotion/react@'))).toBe(true)
    const dns = notices.find((n) => n.name.startsWith('github.com/miekg/dns@'))
    expect(dns?.texts.join('\n')).toMatch(/redistribution and use in source and binary forms/i)
    const react = notices.find((n) => n.name.startsWith('react@'))
    expect(react?.texts.join('\n')).toMatch(/permission is hereby granted/i)
  })
})
