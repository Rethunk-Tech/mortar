import { useSecondValue } from '@rethunk/react-load-hooks'
import { formatElapsed } from '../i18n/formatElapsed.ts'

/** The time since `since`, ticking once a second; only this text redraws. */
export function Elapsed({ since }: { since: number }) {
  return <>{useSecondValue((now) => formatElapsed(now - since), true)}</>
}
