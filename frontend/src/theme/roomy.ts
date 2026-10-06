import { create } from 'zustand'

// Touch or a gamepad as the main input gets larger targets: touch from the pointer media query at startup, a pad from
// its first press for the rest of the session (shell/gamepad).
export const useRoomy = create<{ roomy: boolean }>(() => ({
  roomy: typeof matchMedia === 'function' && matchMedia('(pointer: coarse)').matches,
}))
