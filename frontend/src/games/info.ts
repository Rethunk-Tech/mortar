import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { currentGame, useCurrentGame } from '../nav/currentGame.ts'
import { useProfiles } from '../profiles/store.ts'

const useGameList = create<{ games: GameInfo[] }>(() => ({ games: [] }))
let requested = false

// The game list loads on first lookup, and again when a catalog fetched after start replaces the startup one.
function ensureLoaded() {
  if (requested) {
    return
  }
  requested = true
  List()
    .then((games) => useGameList.setState({ games: games ?? [] }))
    .catch(() => {
      requested = false
    })
}

function find(games: GameInfo[], loaded: GameInfo | null, id: string): GameInfo | undefined {
  return games.find((g) => g.id === id) ?? (loaded?.id === id ? loaded : undefined)
}

// Used when the catalog has not answered yet, so sentences still read.
const FALLBACK_NAME = 'the game'

export function gameInfo(id = currentGame()): GameInfo | undefined {
  ensureLoaded()
  return find(useGameList.getState().games, useProfiles.getState().game, id)
}

export function gameName(id?: string): string {
  return gameInfo(id)?.name || FALLBACK_NAME
}

export function useGameInfo(id?: string): GameInfo | undefined {
  ensureLoaded()
  const current = useCurrentGame()
  const games = useGameList((s) => s.games)
  const loaded = useProfiles((s) => s.game)
  return find(games, loaded, id ?? current)
}

// The catalog's games, [] until it answers.
export function useGames(): GameInfo[] {
  ensureLoaded()
  return useGameList((s) => s.games)
}

export function useGameLoader(id?: string): string {
  return useGameInfo(id)?.loader ?? ''
}

export function useGameName(id?: string): string {
  return useGameInfo(id)?.name || FALLBACK_NAME
}

/** Reads the catalog again, after a game was added or removed. */
export function reloadGames() {
  requested = false
  ensureLoaded()
}

let listening = false

export function initGameList() {
  if (listening || typeof Events.On !== 'function') {
    return
  }
  listening = true
  Events.On('catalog:changed', reloadGames)
}
