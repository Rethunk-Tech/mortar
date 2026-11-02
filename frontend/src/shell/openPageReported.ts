import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
import { reportError } from '../toasts/report.ts'
import { openWeb } from './openWeb.ts'

/** Opens a page in the browser, naming the failure when it cannot. */
export const openPageReported = (url: string): Promise<void> =>
  openWeb(url).catch(reportError(i18n._(msg`Could not open the page`)))
