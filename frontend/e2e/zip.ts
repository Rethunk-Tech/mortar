import { execFileSync } from 'node:child_process'

/** Zips paths (files or folders, relative to cwd) into zip. Runs under Node and Bun: Playwright specs import it. */
export function zipPaths(zip: string, paths: string[], cwd?: string): void {
  execFileSync('python3', ['-m', 'zipfile', '-c', zip, ...paths], { cwd })
}
