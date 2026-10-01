import { create } from 'zustand'
import type { Tool } from '../../bindings/github.com/Rethunk-AI/mortar/internal/tools/models.ts'
import {
  Add,
  Launch,
  List,
  Remove,
  Update,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/tools/service.ts'

interface State {
  tools: Tool[]
  loadedGame: string
  load: (game: string) => Promise<void>
  add: (game: string, tool: Tool) => Promise<Tool>
  update: (game: string, tool: Tool) => Promise<void>
  remove: (game: string, id: string) => Promise<void>
  launch: (game: string, profileID: string, id: string) => Promise<void>
}

export const useTools = create<State>((set, get) => ({
  tools: [],
  loadedGame: '',
  load: async (game) => {
    const tools = await List(game)
    set({ tools, loadedGame: game })
  },
  add: async (game, tool) => {
    const saved = await Add(game, tool)
    if (get().loadedGame === game) {
      set({ tools: [...get().tools, saved] })
    }
    return saved
  },
  update: async (game, tool) => {
    await Update(game, tool)
    if (get().loadedGame === game) {
      set({
        tools: get().tools.map((t) => (t.id === tool.id ? tool : t)),
      })
    }
  },
  remove: async (game, id) => {
    await Remove(game, id)
    if (get().loadedGame === game) {
      set({ tools: get().tools.filter((t) => t.id !== id) })
    }
  },
  launch: (game, profileID, id) => Launch(game, profileID, id),
}))
