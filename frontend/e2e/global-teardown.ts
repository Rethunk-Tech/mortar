import { selftest } from './sandbox.ts'

export default function globalTeardown() {
  selftest('stop')
}
