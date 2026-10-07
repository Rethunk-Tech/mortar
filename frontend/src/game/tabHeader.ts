// The first non-empty line of a profile's notes, for the one-line strip in the tab header.
export function notesFirstLine(notes: string): string {
  return (
    notes
      .split('\n')
      .map((line) => line.trim())
      .find((line) => line !== '') ?? ''
  )
}
