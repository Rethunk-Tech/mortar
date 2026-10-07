// A card's thumbnail is read through Mortar's picture cache, which keeps it on disk and answers from there each time
// Browse is opened again; a CDN asked for the same twenty pictures on every visit drops some of them.
function pictureSrc(picture: string): string {
  return `/mod-picture/?u=${encodeURIComponent(picture)}`
}

export { pictureSrc }
