import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type {
  Entry,
  Level,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import {
  type Lines as Batch,
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import {
  Lines,
  RunLines,
  Send,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { useSettings } from '../settings/store.ts'
import { errorDetails, errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { DEFAULT_FILTERS, type Filters, levelsFromFloor } from './filter.ts'
import { pushCommand } from './history.ts'

function consoleDefaults() {
  const s = useSettings.getState()
  return {
    filters: { ...DEFAULT_FILTERS, levels: levelsFromFloor(s.consoleLevel || 'info') },
    timestamps: s.consoleTimestamps !== false,
    follow: s.consoleFollow !== false,
  }
}

const lastSeq = (entries: Entry[]) => entries.at(-1)?.seq ?? 0

export function canSendTo(status: Status | null, game: string, openId: string) {
  return status?.state === State.Running && status.game === game && status.profile === openId
}

export const useConsole = create<{
  // The game and profile whose log is shown; lines of any other profile are ignored.
  shown: { game: string; profile: string }
  entries: Entry[]
  filters: Filters
  timestamps: boolean
  follow: boolean
  // Lines up to this seq were cleared from the view and stay hidden when the history is read again.
  cleared: number
  // Commands sent per game, oldest first, kept in memory only.
  history: Record<string, string[]>
  // Each request carries a fresh n so jumping to the same row twice scrolls twice.
  jump: { index: number; n: number } | null
  // The Get help dialog, opened from the Console tab or the sidebar's Support menu.
  helping: boolean
  // A recorded run's id, or empty while showing the live session.
  viewingRun: string
  add: (batch: Batch) => void
  reset: (game: string, profile: string) => void
  clear: () => void
  jumpTo: (index: number) => void
  load: (game: string, profile: string) => Promise<void>
  viewRun: (game: string, profile: string, runId: string) => void
  send: (game: string, command: string) => Promise<boolean>
  setSearch: (search: string) => void
  toggleLevel: (level: Level) => void
  setMods: (mods: string[]) => void
  setExcludeMods: (mods: string[]) => void
  clearFilters: () => void
  setTimestamps: (on: boolean) => void
  setFollow: (on: boolean) => void
  setHelping: (on: boolean) => void
}>((set, get) => {
  const initial = consoleDefaults()
  return {
    shown: { game: '', profile: '' },
    entries: [],
    filters: initial.filters,
    timestamps: initial.timestamps,
    follow: initial.follow,
    cleared: 0,
    history: {},
    jump: null,
    helping: false,
    viewingRun: '',
    add: ({ game, profile, entries }) => {
      const { shown, viewingRun } = get()
      if (
        viewingRun !== '' ||
        game !== shown.game ||
        (profile !== '' && profile !== shown.profile)
      ) {
        return
      }
      const seen = Math.max(lastSeq(get().entries), get().cleared)
      const fresh = (entries ?? []).filter((e) => e.seq > seen)
      if (fresh.length > 0) {
        set((s) => ({ entries: [...s.entries, ...fresh] }))
      }
    },
    reset: (game, profile) =>
      set({
        shown: { game, profile },
        entries: [],
        cleared: 0,
        jump: null,
        follow: consoleDefaults().follow,
        viewingRun: '',
      }),
    clear: () =>
      set((s) => ({ entries: [], jump: null, cleared: Math.max(s.cleared, lastSeq(s.entries)) })),
    jumpTo: (index) => set((s) => ({ follow: false, jump: { index, n: (s.jump?.n ?? 0) + 1 } })),
    // Showing another profile starts over; lines that arrived while the history was loading are newer than it and
    // are kept.
    load: async (game, profile) => {
      const { shown } = get()
      if (game !== shown.game || profile !== shown.profile) {
        get().reset(game, profile)
      }
      // On a fresh start the Console can mount before the open profile is known; there is no log to ask for yet.
      if (!profile) {
        return
      }
      try {
        const { viewingRun } = get()
        const lines =
          (viewingRun === ''
            ? await Lines(game, profile)
            : await RunLines(game, profile, viewingRun)) ?? []
        const now = get().shown
        if (now.game !== game || now.profile !== profile || get().viewingRun !== viewingRun) {
          return
        }
        if (viewingRun !== '') {
          set({ entries: lines })
          return
        }
        const history = lines.filter((e) => e.seq > get().cleared)
        const after = lastSeq(history)
        set((s) => ({ entries: [...history, ...s.entries.filter((e) => e.seq > after)] }))
      } catch (e) {
        useToasts.getState().push({
          kind: 'error',
          title: i18n._(msg`Could not read the SMAPI log`),
          body: errorMessage(e),
          detail: errorDetails(e),
        })
      }
    },
    viewRun: (game, profile, runId) => {
      set({
        shown: { game, profile },
        viewingRun: runId,
        entries: [],
        cleared: 0,
        jump: null,
        follow: runId === '',
      })
      get()
        .load(game, profile)
        .then(() => undefined)
    },
    // Sending brings back the tail so the command's output scrolls into view.
    send: async (game, command) => {
      try {
        await Send(game, command)
      } catch (e) {
        useToasts.getState().push({
          kind: 'error',
          title: i18n._(msg`Could not run the command`),
          body: errorMessage(e),
          detail: errorDetails(e),
        })
        return false
      }
      set((s) => ({
        follow: true,
        history: { ...s.history, [game]: pushCommand(s.history[game] ?? [], command) },
      }))
      return true
    },
    setSearch: (search) => set((s) => ({ filters: { ...s.filters, search } })),
    toggleLevel: (level) =>
      set((s) => ({
        filters: {
          ...s.filters,
          levels: s.filters.levels.includes(level)
            ? s.filters.levels.filter((l) => l !== level)
            : [...s.filters.levels, level],
        },
      })),
    setMods: (mods) => set((s) => ({ filters: { ...s.filters, mods } })),
    setExcludeMods: (excludeMods) => set((s) => ({ filters: { ...s.filters, excludeMods } })),
    clearFilters: () => set({ filters: consoleDefaults().filters }),
    setTimestamps: (timestamps) => set({ timestamps }),
    setFollow: (follow) => set({ follow }),
    setHelping: (helping) => set({ helping }),
  }
})
