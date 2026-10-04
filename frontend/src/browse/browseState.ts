const DEBOUNCE_MS = 300
const PAGE_SIZE = 20
const FIRST_PAGE = 1

interface Paging {
  page: number
  total: number
}

function debounceDue(elapsedMs: number): boolean {
  return elapsedMs >= DEBOUNCE_MS
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

function nextPage(paging: Paging): number {
  return clampPage({ page: paging.page + FIRST_PAGE, total: paging.total })
}

function prevPage(paging: Paging): number {
  return clampPage({ page: paging.page - FIRST_PAGE, total: paging.total })
}

export { clampPage, DEBOUNCE_MS, debounceDue, nextPage, PAGE_SIZE, prevPage }
