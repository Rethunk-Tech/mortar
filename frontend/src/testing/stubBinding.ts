import { spyOn } from 'bun:test'
import { CancellablePromise } from '@wailsio/runtime'

// Replaces one Wails binding until the test file's afterEach runs mock.restore(). Unlike mock.module, the stub never
// outlives the test or hides the module's other exports from later files. `answer` may return a value or a promise,
// or throw to reject.
export function stubBinding<M extends object, K extends keyof M>(
  service: M,
  name: K,
  answer: (...args: never[]) => unknown,
) {
  return spyOn(service, name).mockImplementation(
    ((...args: never[]) => new CancellablePromise((resolve) => resolve(answer(...args)))) as never,
  )
}
