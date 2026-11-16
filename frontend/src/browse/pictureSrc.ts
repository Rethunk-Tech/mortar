// Hosts Mortar will fetch a picture from (the same CDNs internal/modpic allows). A README image on another host
// is dropped rather than loaded in the WebView.
const allowedHosts = new Set([
  'staticdelivery.nexusmods.com',
  'images.nexusmods.com',
  'ccdn.thunderstore.io',
  'gcdn.thunderstore.io',
  'cdn.modrinth.com',
  'media.forgecdn.net',
  'avatars.githubusercontent.com',
  'img.itch.zone',
  'thumb.modcdn.io',
  'moddbcdn.vintagestory.at',
  'modscontent.paradox-interactive.com',
])

// A card's thumbnail is read through Mortar's picture cache, which keeps it on disk and answers from there each time
// Browse is opened again; a CDN asked for the same twenty pictures on every visit drops some of them.
function pictureSrc(picture: string): string {
  return `/mod-picture/?u=${encodeURIComponent(picture)}`
}

function cachedPictureSrc(picture: string): string | undefined {
  let host: string
  try {
    const u = new URL(picture)
    if (
      u.protocol !== 'https:' ||
      u.username ||
      u.password ||
      u.pathname === '' ||
      u.pathname === '/'
    ) {
      return
    }
    host = u.hostname.toLowerCase()
  } catch {
    return
  }
  if (!allowedHosts.has(host)) {
    return
  }
  return pictureSrc(picture)
}

export { cachedPictureSrc, pictureSrc }
