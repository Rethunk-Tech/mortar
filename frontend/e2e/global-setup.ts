import { freshSandbox } from './sandbox.ts'

// The returned function is Playwright's global teardown, so it removes the sandbox this setup made and no other.
export default function globalSetup() {
  return freshSandbox()
}
