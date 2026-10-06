import { create } from 'zustand'

// Play mode: Mortar was started by a shortcut, usually from Steam's Game Mode, to play one profile (launch/playMode).
// cancelled is set when the blocked-Play dialog was cancelled, which quits rather than opening the full window.
const usePlayMode = create<{ solo: boolean; cancelled: boolean }>(() => ({
  solo: false,
  cancelled: false,
}))

/** Wraps a launch dialog's Cancel, so in play mode it quits instead of opening the full window. */
function cancelling(fn: () => void) {
  return () => {
    usePlayMode.setState({ cancelled: usePlayMode.getState().solo })
    fn()
  }
}

export { cancelling, usePlayMode }
