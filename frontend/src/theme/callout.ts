import { alpha, type Theme } from '@mui/material'

type Tone = 'warning' | 'info' | 'success'

const FILL = 0.14
const LINE = 0.5

// The tinted fill and outline of a box that asks for attention, the same in every view.
export const calloutFill = (tone: Tone) => (th: Theme) => alpha(th.palette[tone].main, FILL)
export const calloutLine = (tone: Tone) => (th: Theme) => alpha(th.palette[tone].main, LINE)
