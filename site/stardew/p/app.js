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

function row(e) {
  const li = el('li')
  if (e.kind === 'nexus') {
    li.append(
      el(
        'a',
        {
          href: `https://www.nexusmods.com/stardewvalley/mods/${e.mod}`,
          rel: 'noopener noreferrer',
        },
        `Nexus mod ${e.mod}`,
      ),
    )
  } else {
    li.append(
      el(
        'a',
        {
          href: `https://github.com/${e.repo}/releases/tag/${encodeURIComponent(e.tag)}`,
          rel: 'noopener noreferrer',
        },
        `${e.repo} ${e.tag}`,
      ),
    )
  }
  return li
}

async function main() {
  const link = location.href
  const payload = location.hash.replace(HASH, '')
  let share
  try {
    share = await decodeShare(location.hash)
  } catch (err) {
    const kind = err?.kind
    show(
      kind === 'missing'
        ? 'This link has no profile in it. Ask the sender for the full link.'
        : kind === 'version'
          ? 'This profile was shared by a newer Mortar. Update Mortar, then open the link again.'
          : 'This link is damaged or too large to read. Ask the sender to share it again.',
    )
    return
  }
  $('name').textContent = share.name
  $('count').textContent = `${share.entries.length} ${share.entries.length === 1 ? 'mod' : 'mods'}`
  $('list').append(...share.entries.map(row))
  $('profile').hidden = false
  $('open').href = `mortar://stardew/p/${payload}`
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
