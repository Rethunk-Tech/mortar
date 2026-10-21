import { visitorOS } from './os.ts'

const fedoraRe = /Fedora/i
const debianRe = /Ubuntu|Debian/i
const archRe = /Arch Linux|archlinux/i
const versionRe = /VERSION/g
const leadingVRe = /^v/
const copiedMs = 1500
const kib = 1024
const mib = kib * kib

function mortarSelect(tablist, id) {
  for (const tab of tablist.querySelectorAll(':scope > [role="tab"]')) {
    const on = tab.getAttribute('aria-controls') === id
    tab.setAttribute('aria-selected', String(on))
    tab.tabIndex = on ? 0 : -1
    document.getElementById(tab.getAttribute('aria-controls')).hidden = !on
  }
  tablist.dispatchEvent(new CustomEvent('mortar-select', { detail: id }))
}

function mortarTabs(tablist, initial) {
  mortarSelect(tablist, initial)
  tablist.addEventListener('click', (e) => {
    const tab = e.target.closest('[role="tab"]')
    if (tab) {
      mortarSelect(tablist, tab.getAttribute('aria-controls'))
      history.replaceState(null, '', `#${tab.getAttribute('aria-controls')}`)
    }
  })
  tablist.addEventListener('keydown', (e) => {
    if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') {
      return
    }
    const tabs = [...tablist.querySelectorAll(':scope > [role="tab"]')]
    const i = tabs.indexOf(document.activeElement)
    const next = tabs[(i + (e.key === 'ArrowRight' ? 1 : tabs.length - 1)) % tabs.length]
    mortarSelect(tablist, next.getAttribute('aria-controls'))
    next.focus()
  })
}

function mortarCopyButtons() {
  for (const pre of document.querySelectorAll('.panel pre')) {
    const wrap = document.createElement('div')
    wrap.className = 'code'
    pre.replaceWith(wrap)
    wrap.append(pre)
    const button = document.createElement('button')
    button.type = 'button'
    button.className = 'copy-btn'
    button.textContent = 'Copy'
    button.addEventListener('click', () => {
      navigator.clipboard.writeText(pre.textContent).then(() => {
        button.textContent = 'Copied'
        setTimeout(() => {
          button.textContent = 'Copy'
        }, copiedMs)
      })
    })
    wrap.append(button)
  }
}

function mortarSize(bytes) {
  return bytes >= mib ? `${(bytes / mib).toFixed(1)} MB` : `${Math.round(bytes / kib)} KB`
}

const mortarVersion = document.querySelector('.dl')?.dataset.version ?? ''

function mortarRelease(rel) {
  const version = rel.tag_name.replace(leadingVRe, '')
  document.getElementById('version').textContent = `Mortar ${rel.tag_name}`
  const assets = new Map(rel.assets.map((a) => [a.name, a]))
  for (const link of document.querySelectorAll('[data-asset]')) {
    const asset = assets.get(link.dataset.asset.replace(versionRe, version))
    if (asset) {
      link.href = asset.browser_download_url
      if (link.classList.contains('file')) {
        const size = document.createElement('span')
        size.className = 'size'
        size.textContent = mortarSize(asset.size)
        link.append(size)
      }
    }
  }
}

;(() => {
  document.documentElement.classList.add('js')
  const ua = navigator.userAgent
  const hash = location.hash.slice(1)
  const distroIds = ['deb', 'fedora', 'arch', 'flatpak', 'brew']
  const modeIds = ['linux-store', 'linux-files', 'linux-cli']
  let distro = 'deb'
  if (distroIds.includes(hash)) {
    distro = hash
  } else if (fedoraRe.test(ua)) {
    distro = 'fedora'
  } else if (archRe.test(ua) && !debianRe.test(ua)) {
    distro = 'arch'
  }
  let mode = 'linux-store'
  if (modeIds.includes(hash)) {
    mode = hash
  } else if (distroIds.includes(hash)) {
    mode = 'linux-cli'
  } else if (hash === 'appimage') {
    mode = 'linux-files'
  }
  const os = visitorOS()
  let platform = os ?? 'windows'
  if (
    distroIds.includes(hash) ||
    modeIds.includes(hash) ||
    hash === 'appimage' ||
    hash === 'linux'
  ) {
    platform = 'linux'
  } else if (hash === 'windows' || hash === 'deck' || hash === 'extension') {
    platform = hash
  }

  const systems = document.getElementById('systems')
  const other = document.getElementById('other')
  const sysname = document.getElementById('sysname')
  const verify = document.getElementById('verify')
  // A system Mortar has no build for sees every choice instead of a guess.
  systems.hidden = os !== null
  other.parentElement.hidden = os === null
  other.addEventListener('click', () => {
    systems.hidden = !systems.hidden
    other.setAttribute('aria-expanded', String(!systems.hidden))
  })
  systems.addEventListener('mortar-select', (e) => {
    sysname.textContent = document.getElementById(`tab-${e.detail}`).textContent.trim()
    const commands = e.detail === 'windows' ? 'windows' : 'linux'
    verify.hidden = e.detail === 'extension'
    for (const block of verify.querySelectorAll('[data-for]')) {
      block.hidden = block.dataset.for !== commands
    }
  })
  // The verify commands name the file the visitor last picked.
  document.addEventListener('click', (e) => {
    const link = e.target.closest('.file[data-asset]')
    if (!link) {
      return
    }
    const name = link.dataset.asset.replace(versionRe, mortarVersion)
    for (const f of verify.querySelectorAll('[data-for]:not([hidden]) .vfile')) {
      f.textContent = name
    }
  })

  mortarTabs(systems, platform)
  mortarTabs(document.querySelector('.tabs.modes'), mode)
  mortarTabs(document.querySelector('.tabs.sub'), distro)
  mortarCopyButtons()
  fetch('https://api.github.com/repos/Rethunk-Tech/mortar/releases/latest')
    .then((r) => (r.ok ? r.json() : null))
    .then((rel) => {
      if (rel?.tag_name) {
        mortarRelease(rel)
      }
    })
    .catch(() => null)
})()
