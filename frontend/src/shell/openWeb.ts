import { OpenWeb } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import { errorDetails } from '../toasts/errorKind.ts'

// opener.ErrOpenInPage: a server build (the self-test sandbox) cannot start the browser of the person using it, so it
// hands the address back and this tab's browser opens it.
const OPEN_IN_PAGE = 'mortar:open-in-page'

/** Opens an http or https address in the default browser, the way the backend's opener allows. */
export async function openWeb(url: string): Promise<void> {
  try {
    await OpenWeb(url)
  } catch (e) {
    if (!errorDetails(e).includes(OPEN_IN_PAGE)) {
      throw e
    }
    window.open(url, '_blank', 'noopener,noreferrer')
  }
}
