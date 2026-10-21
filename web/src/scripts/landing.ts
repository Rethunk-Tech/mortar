// The landing page: the download button names the visitor's system, the metrics count up once, the game picker
// searches as you type ("/" jumps to it), and the demo preview loops until a click swaps in the real app.
import { visitorOS } from './os.ts'
import { track } from './track.ts'

const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
const PICKER_TILES = 8
const COUNT_MS = 1200
const EASE_POWER = 3
const LOOP_MS = 3500

function systemButton() {
  const cta = document.getElementById('ctaos')
  const os = visitorOS()
  if (cta && os) {
    cta.innerHTML = `<svg aria-hidden="true"><use href="/img/brands.svg#${os}"></use></svg>Download for ${os === 'linux' ? 'Linux' : 'Windows'}`
    cta.setAttribute('href', `/download/#${os}`)
  }
}

function countUp() {
  const metrics = document.querySelector('.metrics')
  const els = [...document.querySelectorAll<HTMLElement>('.metrics [data-count]')]
  if (!metrics || reduce) {
    return
  }
  const targets = els.map((el) => Number(el.dataset.count))
  new IntersectionObserver(
    (entries, observer) => {
      if (!entries.some((e) => e.isIntersecting)) {
        return
      }
      observer.disconnect()
      const t0 = performance.now()
      const step = (t: number) => {
        const k = Math.min(1, (t - t0) / COUNT_MS)
        const eased = 1 - (1 - k) ** EASE_POWER
        els.forEach((el, i) => {
          el.textContent = String(Math.round((targets[i] ?? 0) * eased))
        })
        if (k < 1) {
          requestAnimationFrame(step)
        }
      }
      requestAnimationFrame(step)
    },
    { threshold: 0.5 },
  ).observe(metrics)
}

function picker() {
  const input = document.querySelector<HTMLInputElement>('.searchbox input')
  const tiles = document.getElementById('tiles')
  const matches = document.getElementById('matches')
  const data = document.getElementById('picker-games')?.textContent
  if (!(input && tiles && matches && data)) {
    return
  }
  const games = JSON.parse(data) as [id: string, name: string, capsule: string][]
  const popular = tiles.innerHTML
  const tile = ([id, name, capsule]: [string, string, string]) => {
    const li = document.createElement('li')
    const a = document.createElement('a')
    a.href = `/${id}/`
    if (capsule) {
      const img = document.createElement('img')
      Object.assign(img, {
        src: capsule,
        alt: '',
        width: 300,
        height: 450,
        referrerPolicy: 'no-referrer',
      })
      a.append(img)
    } else {
      const letter = document.createElement('b')
      letter.className = 'letter'
      letter.textContent = name.charAt(0)
      a.append(letter)
    }
    const label = document.createElement('span')
    label.textContent = name
    a.append(label)
    li.append(a)
    return li
  }
  input.addEventListener('input', () => {
    const q = input.value.trim().toLowerCase()
    if (!q) {
      tiles.innerHTML = popular
      matches.textContent = matches.dataset.idle ?? ''
      return
    }
    const hits = games.filter(([, name]) => name.toLowerCase().includes(q))
    tiles.replaceChildren(...hits.slice(0, PICKER_TILES).map(tile))
    matches.textContent =
      hits.length === 0
        ? 'No game by that name.'
        : `${hits.length} ${hits.length === 1 ? 'match' : 'matches'}${hits.length > PICKER_TILES ? `; showing the first ${PICKER_TILES}` : ''}`
  })
  globalThis.addEventListener('keydown', (e: KeyboardEvent) => {
    const t = e.target
    if (
      e.key !== '/' ||
      e.ctrlKey ||
      e.metaKey ||
      e.altKey ||
      (t instanceof Element && t.closest('input,textarea,select'))
    ) {
      return
    }
    e.preventDefault()
    input.scrollIntoView({ block: 'center' })
    input.focus({ preventScroll: true })
  })
}

function demo() {
  const frames = [...document.querySelectorAll<HTMLImageElement>('.loop img')]
  let at = 0
  const timer = reduce
    ? 0
    : setInterval(() => {
        frames[at]?.classList.remove('on')
        at = (at + 1) % frames.length
        frames[at]?.classList.add('on')
      }, LOOP_MS)
  document.getElementById('demo-open')?.addEventListener('click', (e) => {
    clearInterval(timer)
    track('Demo: Start')
    const frame = document.createElement('iframe')
    frame.className = 'demo-frame'
    frame.src = '/demo/'
    frame.title = 'Mortar demo: a sample Lethal Company profile'
    ;(e.currentTarget as HTMLElement).replaceWith(frame)
    frame.focus()
  })
}

export function initLanding() {
  systemButton()
  countUp()
  picker()
  demo()
}
