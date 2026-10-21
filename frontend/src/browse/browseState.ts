import { FIRST_PAGE } from './browseConstants.ts'

const DEBOUNCE_MS = 300
// The hits one source page holds; a screen of results is cut from these.
const PAGE_SIZE = 20

interface Paging {
  page: number
  total: number
  size?: number
}

function clampPage({ page, total, size = PAGE_SIZE }: Paging): number {
  const last = Math.max(FIRST_PAGE, Math.ceil(total / size) || FIRST_PAGE)
  if (page < FIRST_PAGE) {
    return FIRST_PAGE
  }
  if (page > last) {
    return last
  }
  return page
}

// A screen holds whole rows of what fits, and never fewer than one source page.
function screenSize(columns: number, rows: number): number {
  return columns * Math.max(rows, Math.ceil(PAGE_SIZE / columns))
}

// The source pages one screen spans, and how many hits of the first it skips.
function screenSpan(page: number, size: number): { first: number; last: number; skip: number } {
  const start = (page - FIRST_PAGE) * size
  return {
    first: Math.floor(start / PAGE_SIZE) + FIRST_PAGE,
    last: Math.ceil((start + size) / PAGE_SIZE),
    skip: start % PAGE_SIZE,
  }
}

// The screen that starts with the same hit after the screen size changes from one size to another.
function pageKeeping(page: number, from: number, to: number): number {
  return Math.floor(((page - FIRST_PAGE) * from) / to) + FIRST_PAGE
}

export { clampPage, DEBOUNCE_MS, PAGE_SIZE, pageKeeping, screenSize, screenSpan }
