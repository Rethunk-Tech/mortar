import type { Reporter, TestCase, TestResult } from '@playwright/test/reporter'

/** Tells the global teardown, which runs in this process, that a test failed so it keeps the sandbox's logs. */
export default class FailureReporter implements Reporter {
  onTestEnd(_test: TestCase, result: TestResult) {
    if (result.status === 'failed' || result.status === 'timedOut') {
      process.env.MORTAR_E2E_FAILED = '1'
    }
  }
}
