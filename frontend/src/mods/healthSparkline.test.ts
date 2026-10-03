import { describe, expect, test } from 'bun:test'
import {
  buildHealthSparklinePath,
  HEALTH_SPARKLINE_HEIGHT,
  HEALTH_SPARKLINE_WIDTH,
  sparklineSeries,
} from './healthSparkline.ts'

describe('healthSparkline', () => {
  test('sparklineSeries sums problems and updates and caps at 30 points', () => {
    const points = Array.from({ length: 40 }, (_, i) => ({ problems: i, updates: 1 }))
    expect(sparklineSeries(points)).toEqual(Array.from({ length: 30 }, (_, i) => i + 10 + 1))
  })

  test('buildHealthSparklinePath returns empty for no data', () => {
    expect(buildHealthSparklinePath([])).toBe('')
  })

  test('buildHealthSparklinePath scales to width and height', () => {
    const path = buildHealthSparklinePath(
      [0, 2, 4],
      HEALTH_SPARKLINE_WIDTH,
      HEALTH_SPARKLINE_HEIGHT,
    )
    expect(path.startsWith('M')).toBe(true)
    expect(path).toContain('L')
    expect(path).toContain(`${HEALTH_SPARKLINE_WIDTH - 2}`)
  })
})
