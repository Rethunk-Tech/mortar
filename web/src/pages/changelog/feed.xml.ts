// An Atom feed of the releases, one entry each, at the address the app and readers already follow.
import { releases } from '../../lib/changelog.ts'
import { escapeHTML, SITE } from '../../lib/site.ts'

export function GET() {
  const all = releases()
  const entries = all.map(
    (r) => `  <entry>
    <title>Mortar ${r.version}</title>
    <id>${SITE}/changelog/#v${r.version}</id>
    <link href="${SITE}/changelog/#v${r.version}" />
    <updated>${r.date}T00:00:00Z</updated>
    <content type="html">${escapeHTML(r.html)}</content>
  </entry>`,
  )
  const body = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Mortar releases</title>
  <id>${SITE}/changelog/</id>
  <link href="${SITE}/changelog/" />
  <link rel="self" href="${SITE}/changelog/feed.xml" />
  <updated>${all[0]?.date ?? '1970-01-01'}T00:00:00Z</updated>
  <author><name>Rethunk.Tech</name></author>
${entries.join('\n')}
</feed>
`
  return new Response(body, { headers: { 'Content-Type': 'application/atom+xml' } })
}
