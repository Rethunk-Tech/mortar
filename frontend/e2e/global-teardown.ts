import { removeSandbox } from './sandbox.ts'

export default function globalTeardown() {
  removeSandbox()
}
