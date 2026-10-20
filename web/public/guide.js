// Marks the contents link of the guide section being read, so the sticky contents show where you are.
const links = new Map(
  [...document.querySelectorAll('.toc a')].map((a) => [decodeURIComponent(a.hash.slice(1)), a]),
)
const headings = [...document.querySelectorAll('.guide h2[id]')]
let current = null

function mark(id) {
  if (id === current) {
    return
  }
  links.get(current)?.removeAttribute('aria-current')
  links.get(id)?.setAttribute('aria-current', 'true')
  current = id
}

// The last heading above the top quarter of the window is the section being read.
function update() {
  const line = window.innerHeight / 4
  let id = headings[0]?.id ?? null
  for (const h of headings) {
    if (h.getBoundingClientRect().top <= line) {
      id = h.id
    }
  }
  mark(id)
}

addEventListener('scroll', update, { passive: true })
addEventListener('resize', update)
update()
