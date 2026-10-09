import { spawn } from 'node:child_process'
import process from 'node:process'

// Specs mutate one seeded sandbox server, so workers stay at 1 inside a Playwright process. Each shard is its own process
// and therefore its own sandbox (own folder, port and display, see global-setup.ts), so shards run side by side. Two is
// the split of the 49 tests that balances by time (about 60 s each); a third leaves one shard at 50 s.
const SHARDS = 2

function runShard(index: number): Promise<{ code: number; out: string }> {
  const child = spawn(
    'bunx',
    ['playwright', 'test', `--shard=${index}/${SHARDS}`, ...process.argv.slice(2)],
    {
      env: process.env,
    },
  )
  let out = ''
  child.stdout.on('data', (chunk) => {
    out += chunk
  })
  child.stderr.on('data', (chunk) => {
    out += chunk
  })
  return new Promise((done) => child.on('exit', (code) => done({ code: code ?? 1, out })))
}

const results = await Promise.all(Array.from({ length: SHARDS }, (_, i) => runShard(i + 1)))
for (const [i, result] of results.entries()) {
  process.stdout.write(`\n=== shard ${i + 1}/${SHARDS} (exit ${result.code}) ===\n${result.out}`)
}
process.exitCode = results.some((r) => r.code !== 0) ? 1 : 0
