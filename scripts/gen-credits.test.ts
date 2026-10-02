import { describe, expect, test } from 'bun:test'
import { classifyLicenceText } from './gen-credits.ts'

describe('classifyLicenceText', () => {
  test('finds licence text after a copyright header', () => {
    const text = `${'Copyright IBM Corp. 2014, 2026\n'.repeat(300)}
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files.`

    expect(classifyLicenceText(text)).toBe('MIT')
  })
})
