import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import { HealthBadges } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'

interface HealthNotice {
  game: string
  profile: string
  findings: number
}

// What the latest background or manual health check found in each profile of the loaded game; a profile whose
// check came back clean has no entry.
const useHealthBadges = create<{
  game: string
  byProfile: Record<string, number>
  load: (game: string) => void
  apply: (notice: HealthNotice) => void
}>((set, get) => ({
  game: '',
  byProfile: {},
  load: (game) => {
    if (game === '' || get().game === game) {
      return
    }
    listen()
    set({ game, byProfile: {} })
    HealthBadges(game)
      .then((found) => {
        if (get().game === game) {
          set({
            byProfile: Object.fromEntries(
              Object.entries(found ?? {}).map(([id, n]) => [id, n ?? 0]),
            ),
          })
        }
      })
      .catch(() => undefined)
  },
  apply: (notice) => {
    if (notice.game !== get().game) {
      return
    }
    const { [notice.profile]: _, ...rest } = get().byProfile
    set({ byProfile: notice.findings > 0 ? { ...rest, [notice.profile]: notice.findings } : rest })
  },
}))

let listening = false

function listen() {
  if (listening) {
    return
  }
  listening = true
  Events.On('profile:health', (event) =>
    useHealthBadges.getState().apply(event.data as HealthNotice),
  )
}

export { useHealthBadges }
