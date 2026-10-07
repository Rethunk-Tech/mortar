interface GlanceView {
  mods: number
  updates: number
  // Null until the problems are first read, so the line is left out rather than reading "0".
  problems: number | null
}

// Updates read only when there are some; a problem count of zero reads as none.
function glance(mods: number, updates: number, problems: number | null): GlanceView {
  return { mods, updates, problems }
}

const CHANGES_COLLAPSED = 3

function changesView(lines: string[], expanded: boolean) {
  const shown = expanded ? lines : lines.slice(0, CHANGES_COLLAPSED)
  return { shown, canExpand: !expanded && lines.length > CHANGES_COLLAPSED }
}

const SAVES_SHOWN = 4

interface SaveLike {
  farm: string
  folder: string
  year: number
  season: number
  unrecorded: boolean
}

// A save without Stardew's calendar (a Lethal Company slot, a Valheim world) has a name only.
function saveCalendar(save: SaveLike): { year: number; season: number } | null {
  return save.unrecorded || save.year <= 0 ? null : { year: save.year, season: save.season }
}

// A calendar game's save whose SaveGameInfo is missing or empty has no date to show, unlike a game with no calendar.
function saveCalendarUnknown(save: SaveLike): boolean {
  return !save.unrecorded && save.year <= 0
}

function savesView<T extends SaveLike>(saves: T[]) {
  return {
    shown: saves.slice(0, SAVES_SHOWN).map((save) => ({
      save,
      calendar: saveCalendar(save),
    })),
    total: saves.length,
  }
}

// The hero holds the inline name field; with the hero hidden a rename goes to the Edit profile dialog.
function renameSurface(hero: string): 'inline' | 'dialog' {
  return hero === 'hidden' ? 'dialog' : 'inline'
}

export { changesView, glance, renameSurface, saveCalendar, saveCalendarUnknown, savesView }
