const windowsRe = /Windows/i
const linuxRe = /Linux|X11/i
const androidRe = /Android/i
const fedoraRe = /Fedora/i
const debianRe = /Ubuntu|Debian/i
const archRe = /Arch/i
const versionRe = /VERSION/g
const leadingVRe = /^v/
const copiedMs = 1500
const kib = 1024
const mib = kib * kib

function mortarSelect(tablist, id) {
  for (const tab of tablist.querySelectorAll('[role="tab"]')) {
    const on = tab.getAttribute('aria-controls') === id
    tab.setAttribute('aria-selected', String(on))
    tab.tabIndex = on ? 0 : -1
    document.getElementById(tab.getAttribute('aria-controls')).hidden = !on
  }
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
    const tabs = [...tablist.querySelectorAll('[role="tab"]')]
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
    button.className = 'copy'
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
  const platforms = document.querySelector('.tabs:not(.sub)')
  const distros = document.querySelector('.tabs.sub')
  const distroIds = ['deb', 'fedora', 'arch', 'flatpak', 'appimage']
  let distro = 'deb'
  if (distroIds.includes(hash)) {
    distro = hash
  } else if (fedoraRe.test(ua)) {
    distro = 'fedora'
  } else if (archRe.test(ua) && !debianRe.test(ua)) {
    distro = 'arch'
  }
  let platform = 'windows'
  if (distroIds.includes(hash) || hash === 'linux') {
    platform = 'linux'
  } else if (hash === 'windows' || hash === 'extension') {
    platform = hash
  } else if (linuxRe.test(ua) && !androidRe.test(ua) && !windowsRe.test(ua)) {
    platform = 'linux'
  }
  mortarTabs(platforms, platform)
  mortarTabs(distros, distro)
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
