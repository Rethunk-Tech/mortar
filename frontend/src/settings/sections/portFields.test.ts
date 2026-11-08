import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const source = (name: string) => readFileSync(join(import.meta.dir, name), 'utf8')

// A port is an integer in a range, which PrefNumber checks and saves once on Enter or blur.
test('both port fields are PrefNumber, not a hand-made range check', () => {
  for (const name of ['General.tsx', 'StreamOverlay.tsx']) {
    const text = source(name)
    expect(text).toContain('<PrefNumber')
    expect(text).not.toMatch(/\b(commitPort|savePort|portError|portRange)\b/)
  }
})

// A probe that could not read the firewall or network state did not fail to save anything.
test('the firewall and network probes report by cause, not as a save', () => {
  const lines = source('General.tsx').split('\n')
  for (const call of ['FirewallBlocked()', 'NetworkIsPublic()']) {
    const line = lines.find((l) => l.includes(call) && l.includes('.catch(')) ?? ''
    expect(line).toContain('reportUnexpected')
  }
})
