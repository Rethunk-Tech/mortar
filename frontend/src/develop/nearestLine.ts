const REACH = 10

// The line closest to from that has, looking down first (a signature or brace sits just above its code), then up,
// within a few lines; -1 when none is near.
export function nearestLine(from: number, count: number, has: (line: number) => boolean): number {
  for (let step = 0; step <= REACH; step++) {
    for (const n of step === 0 ? [from] : [from + step, from - step]) {
      if (n >= 0 && n < count && has(n)) {
        return n
      }
    }
  }
  return -1
}
