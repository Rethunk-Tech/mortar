import { spawnSync } from 'node:child_process'
import { join } from 'node:path'
import process from 'node:process'

const FIXTURE = join(import.meta.dir, 'fixtures')

/**
 * Serves fixtures/mod-template as the Develop tab's template archive and points the sandbox at it, so the New mod
 * wizard renders without the published template. Returns the server's stop.
 */
export function serveModTemplate(dir: string): () => void {
  const zip = join(dir, 'mod-template.zip')
  const made = spawnSync('zip', ['-q', '-r', zip, 'mod-template'], { cwd: FIXTURE })
  if (made.status !== 0) {
    throw new Error(`zip mod-template: ${made.stderr}`)
  }
  const server = Bun.serve({
    port: 0,
    hostname: '127.0.0.1',
    fetch: () => new Response(Bun.file(zip)),
  })
  process.env.MORTAR_DEV_TEMPLATE = `http://127.0.0.1:${server.port}/mod-template.zip`
  return () => server.stop(true)
}
