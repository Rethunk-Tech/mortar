// Site-wide behaviour: the nav's border, progress line and current section, the narrow-screen menu, sections blurring
// in, feature cards tilting toward the pointer and screenshots opening full size.
const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
const nav = document.querySelector<HTMLElement>('.nav')
const progress = document.getElementById('progress')
const sectionLinks = [...document.querySelectorAll<HTMLAnchorElement>('.nav .links a[data-sec]')]
const sections = sectionLinks.flatMap((a) => document.getElementById(a.dataset.sec ?? '') ?? [])

function onScroll() {
  nav?.classList.toggle('scrolled', scrollY > 10)
  const max = document.documentElement.scrollHeight - innerHeight
  if (progress) {
    progress.style.width = `${max > 0 ? (scrollY / max) * 100 : 0}%`
  }
  let current = ''
  for (const s of sections) {
    if (s.getBoundingClientRect().top < innerHeight / 3) {
      current = s.id
    }
  }
  for (const a of sectionLinks) {
    a.classList.toggle('on', a.dataset.sec === current)
  }
}
addEventListener('scroll', onScroll, { passive: true })
onScroll()

const menu = document.querySelector<HTMLButtonElement>('.nav-menu')
const drawer = document.getElementById('nav-drawer')
function setMenu(open: boolean) {
  if (drawer && menu) {
    drawer.hidden = !open
    menu.setAttribute('aria-expanded', String(open))
  }
}
menu?.addEventListener('click', () => setMenu(drawer?.hidden === true))
document.addEventListener('click', (e) => {
  const t = e.target
  if (drawer && !drawer.hidden && t instanceof Node && !drawer.contains(t) && !menu?.contains(t)) {
    setMenu(false)
  }
})

const reveal = new IntersectionObserver(
  (entries) => {
    for (const e of entries) {
      if (e.isIntersecting) {
        e.target.classList.add('in')
        reveal.unobserve(e.target)
      }
    }
  },
  { threshold: 0.1 },
)
for (const el of document.querySelectorAll('.reveal')) {
  reveal.observe(el)
}

const TILT_DEG = 5
for (const card of document.querySelectorAll<HTMLElement>('.card.tilt')) {
  card.addEventListener('pointermove', (e) => {
    if (reduce) {
      return
    }
    const r = card.getBoundingClientRect()
    const x = (e.clientX - r.left) / r.width - 0.5
    const y = (e.clientY - r.top) / r.height - 0.5
    card.style.transform = `perspective(1000px) rotateY(${x * TILT_DEG}deg) rotateX(${-y * TILT_DEG}deg) translateZ(0)`
  })
  card.addEventListener('pointerleave', () => {
    card.style.transform = ''
  })
}

// A code block that scrolls sideways needs a keyboard stop to scroll it.
for (const pre of document.querySelectorAll('pre')) {
  pre.tabIndex = 0
}

const lightbox = document.getElementById('lightbox')
const big = lightbox?.querySelector('img')
function closeLightbox() {
  lightbox?.classList.remove('open')
  lightbox?.setAttribute('aria-hidden', 'true')
}
document.addEventListener('click', (e) => {
  const t = e.target instanceof Element ? e.target : null
  const img = t?.closest<HTMLImageElement>('img.zoom')
  if (img && lightbox && big) {
    e.preventDefault()
    big.src = img.currentSrc || img.src
    big.alt = img.alt
    lightbox.classList.add('open')
    lightbox.setAttribute('aria-hidden', 'false')
  } else if (t?.closest('#lightbox')) {
    closeLightbox()
  }
})
globalThis.addEventListener('keydown', (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    closeLightbox()
    setMenu(false)
  }
})
