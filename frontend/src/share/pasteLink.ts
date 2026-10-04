type PastedLinkKind = 'share' | 'collection' | 'mod'

const SHARE_HOST = 'mortar.rethunk.tech'
const APP_LINK = 'mortar://'
const NEXUS_HOSTS = new Set(['nexusmods.com', 'www.nexusmods.com', 'next.nexusmods.com'])

const SHARE_PATH = /^\/[^/]+\/p$/
const TRAILING_SLASHES = /\/+$/
const COLLECTION = /^\/games\/[^/]+\/collections\/[^/]+(?:\/revisions\/\d+)?$/
const MOD_ON_GAME = /^\/games\/[^/]+\/mods\/\d+$/
const MOD_ON_DOMAIN = /^\/[^/]+\/mods\/\d+$/

function pasteTargetIsEditable(
  el: {
    tagName?: string
    isContentEditable?: boolean
    closest?: (selector: string) => unknown
  } | null,
): boolean {
  if (!el) {
    return false
  }
  const tag = el.tagName?.toLowerCase()
  if (tag === 'input' || tag === 'textarea' || tag === 'select') {
    return true
  }
  if (el.isContentEditable) {
    return true
  }
  return Boolean(
    el.closest?.('input, textarea, select, [contenteditable=""], [contenteditable="true"]'),
  )
}

function classifyPastedLink(text: string): PastedLinkKind | null {
  const value = text.trim()
  if (!value) {
    return null
  }
  if (value.startsWith(APP_LINK)) {
    return 'share'
  }
  let url: URL
  try {
    url = new URL(value)
  } catch {
    return null
  }
  if (url.protocol !== 'https:') {
    return null
  }
  if (url.hostname === SHARE_HOST && SHARE_PATH.test(url.pathname) && url.hash !== '') {
    return 'share'
  }
  if (!NEXUS_HOSTS.has(url.hostname)) {
    return null
  }
  const path = url.pathname.replace(TRAILING_SLASHES, '') || '/'
  if (COLLECTION.test(path)) {
    return 'collection'
  }
  if (MOD_ON_GAME.test(path) || MOD_ON_DOMAIN.test(path)) {
    return 'mod'
  }
  return null
}

export { classifyPastedLink, pasteTargetIsEditable }
