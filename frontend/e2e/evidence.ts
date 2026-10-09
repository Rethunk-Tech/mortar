import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'

/** What teardown knows about the sandbox's server when it decides whether to keep the sandbox's evidence. */
export interface ServerState {
  /** A test failed in this run. */
  failed: boolean
  /** The recorded server process still runs the sandbox's own binary. */
  alive: boolean
  /** Something still listens on the sandbox's port. */
  listening: boolean
}

/** Keep the logs when a test failed or the server is gone; a clean run with a live server leaves nothing behind. */
export const shouldKeepEvidence = (s: ServerState): boolean => s.failed || !s.alive || !s.listening

/** The reaper's `<pid> <exit code>` lines as sentences; a negative code is the signal number that ended the process. */
export function describeExits(text: string): string[] {
  return text
    .split('\n')
    .filter((line) => line.trim() !== '')
    .map((line) => {
      const [pid = '', code = ''] = line.trim().split(/\s+/)
      const n = Number(code)
      return n < 0 ? `pid ${pid} killed by signal ${-n}` : `pid ${pid} exited with status ${code}`
    })
}

const FILES = ['server.log', 'server.exit', 'home/.local/share/mortar/crash.log']

/** Copies the sandbox's server log, crash log and exit record into `out`, with a summary of why they were kept. */
export function keepEvidence(dir: string, out: string, state: ServerState): void {
  mkdirSync(out, { recursive: true })
  for (const rel of FILES) {
    if (existsSync(`${dir}/${rel}`)) {
      copyFileSync(`${dir}/${rel}`, `${out}/${rel.split('/').pop()}`)
    }
  }
  const exits = existsSync(`${dir}/server.exit`)
    ? describeExits(readFileSync(`${dir}/server.exit`, 'utf8'))
    : ['no exit record (the sandbox has no PID namespace)']
  writeFileSync(
    `${out}/server-state.txt`,
    [
      `test failed: ${state.failed}`,
      `server process alive: ${state.alive}`,
      `port listening: ${state.listening}`,
      ...exits,
      '',
    ].join('\n'),
  )
}
