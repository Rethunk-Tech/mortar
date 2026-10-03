const HEALTH_SPARKLINE_POINTS = 30
const HEALTH_SPARKLINE_WIDTH = 120
const HEALTH_SPARKLINE_HEIGHT = 28
const HEALTH_SPARKLINE_PAD = 2

interface SparklinePoint {
  problems: number
  updates: number
}

function sparklineSeries(points: SparklinePoint[]): number[] {
  const slice = points.slice(-HEALTH_SPARKLINE_POINTS)
  return slice.map((p) => p.problems + p.updates)
}

function buildHealthSparklinePath(
  values: number[],
  width = HEALTH_SPARKLINE_WIDTH,
  height = HEALTH_SPARKLINE_HEIGHT,
): string {
  if (values.length === 0) {
    return ''
  }
  const max = Math.max(1, ...values)
  const innerW = width - HEALTH_SPARKLINE_PAD * 2
  const innerH = height - HEALTH_SPARKLINE_PAD * 2
  const step = values.length > 1 ? innerW / (values.length - 1) : 0
  const coords = values.map((v, i) => {
    const x = HEALTH_SPARKLINE_PAD + i * step
    const y = HEALTH_SPARKLINE_PAD + innerH - (v / max) * innerH
    return `${x.toFixed(1)},${y.toFixed(1)}`
  })
  return `M${coords.join(' L')}`
}

export type { SparklinePoint }
export {
  buildHealthSparklinePath,
  HEALTH_SPARKLINE_HEIGHT,
  HEALTH_SPARKLINE_POINTS,
  HEALTH_SPARKLINE_WIDTH,
  sparklineSeries,
}
