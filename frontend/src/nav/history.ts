import { create } from 'zustand'
import { type TabId, useTab } from '../game/tab.ts'
import { dialogOpen, isTypingTarget } from '../settings/shortcuts.ts'
import { sameValue } from '../shell/sameValue.ts'
import { type Route, useNav } from './store.ts'

interface Visit {
  route: Route
  tab: TabId
}

interface History {
  visits: Visit[]
  at: number
}

const MAX_VISITS = 50

// The tab only tells visits to a game's page apart; other pages are one visit however the tab was left.
function same(a: Visit, b: Visit): boolean {
  const tab = a.route.name !== 'game' || a.tab === b.tab
  return tab && sameValue(a.route, b.route)
}

/** The history after visiting a page: forward visits are dropped, and a repeat of the current page adds nothing. */
function visit(h: History, v: Visit): History {
  const current = h.visits[h.at]
  if (current && same(current, v)) {
    return h
  }
  const visits = [...h.visits.slice(0, h.at + 1), v].slice(-MAX_VISITS)
  return { visits, at: visits.length - 1 }
}

/** The history one visit back or forward (step -1 or 1), unchanged at either end. */
function move(h: History, step: -1 | 1): History {
  const at = h.at + step
  return at < 0 || at >= h.visits.length ? h : { ...h, at }
}

const EMPTY: History = { visits: [], at: -1 }

const useNavHistory = create<History>(() => EMPTY)

let restoring = false

function go(step: -1 | 1) {
  const next = move(useNavHistory.getState(), step)
  const target = next.visits[next.at]
  if (next === useNavHistory.getState() || !target) {
    return
  }
  useNavHistory.setState(next)
  restoring = true
  useNav.setState({ route: target.route })
  useTab.getState().setTab(target.tab)
  restoring = false
}

function record() {
  if (!restoring) {
    useNavHistory.setState((h) =>
      visit(h, { route: useNav.getState().route, tab: useTab.getState().tab }),
    )
  }
}

const BACK_BUTTON = 3
const FORWARD_BUTTON = 4

/** Back and forward through the pages and tabs visited, with the mouse's side buttons and Alt+Left and Alt+Right. */
function initNavHistory() {
  record()
  useNav.subscribe(record)
  useTab.subscribe(record)
  const onMouse = (e: MouseEvent) => {
    if (e.button === BACK_BUTTON || e.button === FORWARD_BUTTON) {
      // The web view would otherwise leave the page.
      e.preventDefault()
      if (e.type === 'mouseup') {
        go(e.button === BACK_BUTTON ? -1 : 1)
      }
    }
  }
  document.addEventListener('mouseup', onMouse)
  document.addEventListener('auxclick', onMouse)
  // Alt+Left is the rebindable back shortcut (settings/useShortcuts.ts), which also keeps it quiet while typing or in a dialog.
  document.addEventListener('keydown', (e) => {
    if (
      e.key === 'ArrowRight' &&
      e.altKey &&
      !e.ctrlKey &&
      !e.metaKey &&
      !e.shiftKey &&
      !isTypingTarget(e.target instanceof HTMLElement ? e.target : null) &&
      !dialogOpen()
    ) {
      e.preventDefault()
      go(1)
    }
  })
}

export { go, type History, initNavHistory, move, visit }
