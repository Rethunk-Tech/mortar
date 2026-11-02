// Site-wide behaviour: the nav's border, progress line and current section, the narrow-screen menu, sections blurring
// in, feature cards tilting toward the pointer and screenshots opening full size.
const PERCENT = 100
const UPPER_THIRD = 3
const CENTRE = 0.5
const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
const nav = document.querySelector<HTMLElement>('.nav')
const progress = document.getElementById('progress')
const sectionLinks = [...document.querySelectorAll<HTMLAnchorElement>('.nav .links a[data-sec]')]
const sections = sectionLinks.flatMap((a) => document.getElementById(a.dataset.sec ?? '') ?? [])

function onScroll() {
  nav?.classList.toggle('scrolled', scrollY > 10)
  const max = document.documentElement.scrollHeight - innerHeight
  if (progress) {
    progress.style.width = `${max > 0 ? (scrollY / max) * PERCENT : 0}%`
  }
  let current = ''
  for (const s of sections) {
    if (s.getBoundingClientRect().top < innerHeight / UPPER_THIRD) {
      current = s.id
    }
  }
  for (const a of sectionLinks) {
    a.classList.toggle('on', a.dataset.sec === current)
  }
}
addEventListener('scroll', onScroll, { passive: true })
onScroll()

// Popover supplies the drawer's light dismiss and Escape; browsers without it get a plain toggle.
const menu = document.querySelector<HTMLButtonElement>('.nav-menu')
const drawer = document.getElementById('nav-drawer')
if (menu && drawer && !('popover' in HTMLElement.prototype)) {
  menu.addEventListener('click', () => {
    menu.setAttribute('aria-expanded', String(drawer.classList.toggle('open')))
  })
}

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
    const x = (e.clientX - r.left) / r.width - CENTRE
    const y = (e.clientY - r.top) / r.height - CENTRE
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

const lightbox = document.querySelector<HTMLDialogElement>('dialog#lightbox')
const big = lightbox?.querySelector('img')
document.addEventListener('click', (e) => {
  const t = e.target instanceof Element ? e.target : null
  const img = t?.closest<HTMLImageElement>('img.zoom')
  if (img && lightbox && big) {
    e.preventDefault()
    big.src = img.currentSrc || img.src
    big.alt = img.alt
    lightbox.showModal()
  } else if (t?.closest('#lightbox')) {
    lightbox?.close()
  }
})
