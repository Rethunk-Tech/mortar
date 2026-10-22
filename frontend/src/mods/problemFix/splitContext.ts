import { createContext, useContext } from 'react'
import type { FixSpec } from './fixActions.tsx'

export type Put = (id: string, spec: FixSpec | null) => void

/** How a FixAction joins the split button of the problem row it renders in. */
export const SplitContext = createContext<Put | null>(null)

/** Whether a component renders inside a problem row's split button, where it declares a FixAction instead. */
export const useInSplitFix = () => useContext(SplitContext) !== null
