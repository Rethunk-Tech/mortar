import { FIRST_PAGE } from './browseConstants.ts'

const DEBOUNCE_MS = 300
const PAGE_SIZE = 20

interface Paging {
  page: number
  total: number
}

function clampPage({ page, total }: Paging): number {
  const last = Math.max(FIRST_PAGE, Math.ceil(total / PAGE_SIZE) || FIRST_PAGE)
  if (page < FIRST_PAGE) {
    return FIRST_PAGE
  }
  if (page > last) {
    return last
  }
  return page
}

export { clampPage, DEBOUNCE_MS, PAGE_SIZE }
