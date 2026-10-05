const windowsRe = /Windows/i
const linuxRe = /Linux|X11/i
const androidRe = /Android/i
;(() => {
  const ua = navigator.userAgent
  const base = 'https://github.com/Rethunk-Tech/mortar/releases/latest/download/'
  const button = document.getElementById('primary-download')
  const label = button.querySelector('span')
  const note = document.getElementById('cta-note')
  if (windowsRe.test(ua)) {
    document.documentElement.classList.add('os-windows')
    button.href = `${base}mortar-amd64-installer.exe`
    label.textContent = 'Download for Windows'
    note.textContent = 'Windows 10 and 11, x64. Other builds below.'
  } else if (linuxRe.test(ua) && !androidRe.test(ua)) {
    document.documentElement.classList.add('os-linux')
    button.href = `${base}mortar-linux-x86_64.AppImage`
    label.textContent = 'Download for Linux'
    note.textContent = 'AppImage for x86_64. Flatpak, .deb, .rpm and Arch below.'
  }
  fetch('https://api.github.com/repos/Rethunk-Tech/mortar/releases/latest')
    .then((r) => (r.ok ? r.json() : null))
    .then((rel) => {
      if (rel?.tag_name) {
        document.getElementById('release').textContent =
          `${rel.tag_name} is out · Free and open source`
      }
    })
    .catch(() => null)
})()
