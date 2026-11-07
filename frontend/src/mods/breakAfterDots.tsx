import { Fragment } from 'react'
import { breakPoints } from './breakPoints.ts'

export function BreakAfterDots({ text }: { text: string }) {
  return breakPoints(text).map(({ at, part }) => (
    <Fragment key={at}>
      {at > 0 ? <wbr /> : null}
      {part}
    </Fragment>
  ))
}
