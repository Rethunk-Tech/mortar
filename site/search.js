// Site search: a palette over the Pagefind index that `bun run site:search` writes beside the pages. The index is
// loaded on first open; a local preview without one says so instead of failing.
const MAX_RESULTS = 8
const DEBOUNCE_MS = 200
const button = document.querySelector('.nav-search')

let dialog
let input
let list
let status
let pagefind
let timer
let run = 0
let active = -1

const FIELDS = new Set(['INPUT', 'TEXTAREA', 'SELECT'])

function typing(el) {
  return el instanceof HTMLElement && (el.isContentEditable || FIELDS.has(el.tagName))
}

function build() {
  dialog = document.createElement('dialog')
  dialog.className = 'search-dialog'
  dialog.setAttribute('aria-label', 'Search')
  dialog.innerHTML =
    '<input type="search" class="search-input" placeholder="Search Mortar" aria-label="Search" role="combobox" aria-expanded="false" aria-controls="search-results" aria-autocomplete="list" autocomplete="off" spellcheck="false" />' +
    '<p class="search-status" aria-live="polite"></p>' +
    '<ul class="search-results" id="search-results" role="listbox" aria-label="Search results"></ul>'
  input = dialog.querySelector('input')
  status = dialog.querySelector('.search-status')
  list = dialog.querySelector('ul')
  dialog.addEventListener('close', () => {
    button.focus()
  })
  dialog.addEventListener('click', (e) => {
    if (e.target === dialog) {
      dialog.close()
    }
  })
  dialog.addEventListener('keydown', onKey)
  input.addEventListener('input', () => {
    clearTimeout(timer)
    timer = setTimeout(() => search(input.value), DEBOUNCE_MS)
  })
  document.body.append(dialog)
}

function show(rows) {
  list.replaceChildren(...rows)
  input.setAttribute('aria-expanded', String(rows.length > 0))
  highlight(rows.length > 0 ? 0 : -1)
}

function highlight(i) {
  active = i
  for (const [n, li] of [...list.children].entries()) {
    li.setAttribute('aria-selected', String(n === i))
  }
  const current = list.children[i]
  if (current) {
    input.setAttribute('aria-activedescendant', current.id)
    current.scrollIntoView({ block: 'nearest' })
  } else {
    input.removeAttribute('aria-activedescendant')
  }
}

function row(result, n) {
  const li = document.createElement('li')
  li.id = `search-result-${n}`
  li.setAttribute('role', 'option')
  const a = document.createElement('a')
  a.href = result.url
  const title = document.createElement('strong')
  title.textContent = result.meta.title ?? result.url
  const excerpt = document.createElement('span')
  // Pagefind escapes everything in an excerpt except the <mark> tags it adds around matches.
  excerpt.innerHTML = result.excerpt
  a.append(title, excerpt)
  li.append(a)
  return li
}

async function load() {
  if (!pagefind) {
    pagefind = await import('/pagefind/pagefind.js')
  }
  return pagefind
}

async function search(query) {
  const term = query.trim()
  run += 1
  const mine = run
  status.textContent = ''
  if (!term) {
    show([])
    return
  }
  try {
    const found = await (await load()).search(term)
    const data = await Promise.all(found.results.slice(0, MAX_RESULTS).map((r) => r.data()))
    if (mine !== run) {
      return
    }
    show(data.map(row))
    status.textContent = data.length > 0 ? '' : 'Nothing matched.'
  } catch {
    if (mine === run) {
      show([])
      status.textContent = 'Search is available on the published site.'
    }
  }
}

function onKey(e) {
  const count = list.children.length
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    if (count > 0) {
      highlight((active + (e.key === 'ArrowDown' ? 1 : count - 1)) % count)
    }
  } else if (e.key === 'Enter') {
    e.preventDefault()
    list.children[active]?.querySelector('a').click()
  }
}

function open() {
  if (!dialog) {
    build()
  }
  if (!dialog.open) {
    dialog.showModal()
    input.select()
    // Warm the index while the visitor types the first letters.
    load().catch(() => {
      status.textContent = 'Search is available on the published site.'
    })
  }
}

button.addEventListener('click', open)
document.addEventListener('keydown', (e) => {
  const chord = (e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k'
  // A page with its own "/" search (the landing page's game picker) keeps the key for it.
  const slash =
    e.key === '/' &&
    !e.ctrlKey &&
    !e.metaKey &&
    !e.altKey &&
    !typing(e.target) &&
    !document.querySelector('[data-slash]')
  if (chord || slash) {
    e.preventDefault()
    open()
  }
})
