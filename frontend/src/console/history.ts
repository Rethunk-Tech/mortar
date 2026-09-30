export const MAX_HISTORY = 100

// Appends a command, oldest first, skipping blanks and a repeat of the last one and keeping the newest MAX_HISTORY.
export function pushCommand(history: string[], command: string): string[] {
  const c = command.trim()
  if (c === '' || history.at(-1) === c) {
    return history
  }
  return [...history, c].slice(-MAX_HISTORY)
}

// Moves the history cursor one command back (-1) or forward (1). A cursor equal to history.length is the
// line being typed, which is where the cursor rests and where Down stops.
export function stepHistory(history: string[], at: number, dir: -1 | 1): number {
  return Math.min(history.length, Math.max(0, at + dir))
}
