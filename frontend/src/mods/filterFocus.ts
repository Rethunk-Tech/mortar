import { createSignal } from '../shell/signal.ts'

const focus = createSignal()

export const requestFilterFocus = focus.emit

export const onFilterFocus = focus.subscribe
