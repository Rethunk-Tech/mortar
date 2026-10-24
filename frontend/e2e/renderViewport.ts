import process from 'node:process'
// The window size every render case starts from: the app's default window, as the site screenshots use.
export const VIEWPORT = { width: 1400, height: 840 }

// Reduced motion unless MORTAR_RENDER_MOTION=1: Mortar follows it, so no ripple or animation is counted.
const MOTION = process.env.MORTAR_RENDER_MOTION === '1' ? 'no-preference' : 'reduce'

// The page every render case runs in.
export const PAGE = { viewport: VIEWPORT, reducedMotion: MOTION } as const
