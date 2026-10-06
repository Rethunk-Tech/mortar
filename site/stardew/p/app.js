import { decodeShare, newer } from './share.js'

const HASH = /^#/
const $ = (id) => document.getElementById(id)

function show(message) {
  $('error').textContent = message
  $('error').hidden = false
}

function el(tag, attrs, text) {
  const n = document.createElement(tag)
  Object.assign(n, attrs)
  if (text) {
    n.textContent = text
  }
  return n
}

const SOURCES = {
  nexus: {
    label: 'Nexus Mods',
    link: (e, keys) => ({
      href: `https://www.nexusmods.com/${encodeURIComponent(keys.nexus)}/mods/${e.mod}`,
      text: `Nexus mod ${e.mod}`,
    }),
  },
  github: {
    label: 'GitHub',
    link: (e) => ({
      href: `https://github.com/${e.repo}/releases/tag/${encodeURIComponent(e.tag)}`,
      text: `${e.repo} ${e.tag}`,
    }),
  },
  thunderstore: {
    label: 'Thunderstore',
    link: (e, keys) => ({
      href: `https://thunderstore.io/c/${encodeURIComponent(keys.thunderstore)}/p/${e.ns}/${e.name}/`,
      text: `${e.ns}/${e.name}`,
    }),
  },
}

// picture is a public, cacheable image for the mod where its coordinates name one: the GitHub owner's avatar, or the
// Thunderstore package icon of the shared version. A Nexus picture needs an API key, so Nexus rows have none.
function picture(e) {
  if (e.kind === 'github') {
    return `https://github.com/${encodeURIComponent(e.repo.split('/')[0])}.png?size=64`
  }
  if (e.kind === 'thunderstore' && e.version) {
    return `https://gcdn.thunderstore.io/live/repository/icons/${e.ns}-${e.name}-${e.version}.png`
  }
  return ''
}

const KIB = 1024
const ONE_DECIMAL_BELOW = 10

function size(kb) {
  let value = kb
  for (const unit of ['KB', 'MB', 'GB']) {
    if (value < KIB || unit === 'GB') {
      return `${value.toFixed(unit === 'KB' || value >= ONE_DECIMAL_BELOW ? 0 : 1)} ${unit}`
    }
    value /= KIB
  }
  return ''
}

function row(e, sourceKeys) {
  const source = SOURCES[e.kind]
  const { href, text } = source.link(e, sourceKeys)
  const li = el('li')
  const src = picture(e)
  const pic = src
    ? el('img', { className: 'pic', src, alt: '', loading: 'lazy', width: 32, height: 32 })
    : el('span', { className: 'pic' })
  if (src) {
    pic.addEventListener('error', () => pic.replaceWith(el('span', { className: 'pic' })))
  }
  const needs = el('span', { className: 'flag', hidden: true })
  li.append(
    pic,
    el('span', { className: 'source' }, source.label),
    el('a', { href, rel: 'noopener noreferrer' }, text),
    el('span', { className: 'size' }, e.kb ? size(e.kb) : ''),
    needs,
  )
  li.dataset.min = e.min ?? ''
  return li
}

// markNeeds marks the rows whose mods need a newer game than version; an empty version flags nothing.
function markNeeds(version) {
  let count = 0
  for (const li of $('list').children) {
    const { min } = li.dataset
    const needs = Boolean(version && min && newer(min, version))
    const f = li.querySelector('.flag')
    f.hidden = !needs
    f.textContent = needs ? `Needs game ${min} or newer` : ''
    count += needs ? 1 : 0
  }
  $('flagged').textContent = count
    ? `${count} ${count === 1 ? 'mod needs' : 'mods need'} a newer game than ${version}.`
    : ''
}

const ERRORS = {
  missing: 'This link has no profile in it. Ask the sender for the full link.',
  version: 'This profile was shared by a newer Mortar. Update Mortar, then open the link again.',
  bad: 'This link is damaged or too large to read. Ask the sender to share it again.',
}

async function main() {
  const link = location.href
  const payload = location.hash.replace(HASH, '')
  let share
  try {
    share = await decodeShare(location.hash)
  } catch (err) {
    const kind = err?.kind
    show(ERRORS[kind] ?? ERRORS.bad)
    return
  }
  $('name').textContent = share.name
  $('count').textContent = `${share.entries.length} ${share.entries.length === 1 ? 'mod' : 'mods'}`
  $('list').append(...share.entries.map((e) => row(e, share.sourceKeys)))
  const total = share.entries.reduce((n, e) => n + (e.kb ?? 0), 0)
  if (total > 0) {
    $('count').textContent += `, about ${size(total)}`
  }
  const version = $('version')
  version.value = share.gameVersion
  version.addEventListener('input', () => markNeeds(version.value.trim()))
  $('versionRow').hidden = !share.entries.some((e) => e.min)
  markNeeds(share.gameVersion)
  $('profile').hidden = false
  $('open').href = `mortar://${share.game}/p/${payload}`
  $('open').hidden = false
  $('download').hidden = false
  $('download').addEventListener('click', () => {
    navigator.clipboard?.writeText(link).then(
      () => {
        $('copied').hidden = false
      },
      () => {
        $('copied').textContent = 'Could not copy the link; copy this page address instead.'
        $('copied').hidden = false
      },
    )
  })
}

main().catch(() => show('Something went wrong reading this link.'))
