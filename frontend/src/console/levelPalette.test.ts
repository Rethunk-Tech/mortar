import { describe, expect, test } from 'bun:test'
import { alpha } from '@mui/material/styles'
import { Level } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { createMortarTheme } from '../theme/theme.ts'
import { levelChrome } from './levelPalette.ts'

describe('levelChrome', () => {
  test('row tints are alpha of warning/error/info so light paper stays readable', () => {
    const light = createMortarTheme('sand', { mode: 'light' })
    const warn = levelChrome(light, Level.Warn)
    const err = levelChrome(light, Level.Error)
    const alert = levelChrome(light, Level.Alert)
    expect(warn.row).toBe(alpha(light.palette.warning.main, 0.08))
    expect(err.row).toBe(alpha(light.palette.error.main, 0.1))
    expect(alert.row).toBe(alpha(light.palette.info.main, 0.1))
    expect(warn.color).toBe(light.palette.warning.main)
    expect(err.color).toBe(light.palette.error.light)
    expect(levelChrome(light, Level.Info).color).toBe(light.palette.text.primary)
  })
})
