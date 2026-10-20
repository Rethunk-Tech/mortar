// Filters the /games/ grid by name, mod loader, mod site and store; every card is in the page, so this only hides.
const grid = document.getElementById('game-grid')
const count = document.getElementById('game-count')
const field = (id) => document.getElementById(id)
const q = field('game-q')
const loader = field('game-loader')
const source = field('game-source')
const store = field('game-store')
const cards = [...grid.children]

function apply() {
  const text = q.value.trim().toLowerCase()
  let shown = 0
  for (const card of cards) {
    const d = card.dataset
    const match =
      (!text || d.name.includes(text)) &&
      (!loader.value || d.loader === loader.value) &&
      (!source.value || d.sources.split('|').includes(source.value)) &&
      (!store.value || d.stores.split('|').includes(store.value))
    card.hidden = !match
    if (match) {
      shown += 1
    }
  }
  count.textContent =
    shown === cards.length ? `${shown} games` : `${shown} of ${cards.length} games`
}

for (const el of [q, loader, source, store]) {
  el.addEventListener('input', apply)
}
apply()
