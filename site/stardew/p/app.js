import { decodeShare } from './share.js'

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

function row(e, sourceKeys) {
  const source = SOURCES[e.kind]
  const { href, text } = source.link(e, sourceKeys)
  const li = el('li')
  li.append(
    el('span', { className: 'source' }, source.label),
    el('a', { href, rel: 'noopener noreferrer' }, text),
  )
  return li
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
