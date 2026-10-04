import { create } from 'zustand'

// The game the Report a bug dialog is open for ('' outside a game's pages), or null while it is closed.
export const useBugReport = create<{ game: string | null }>(() => ({ game: null }))

// Opens Mortar's Report a bug dialog; it prefills a new GitHub issue from what the user writes.
export function reportBug(game: string): void {
  useBugReport.setState({ game })
}
