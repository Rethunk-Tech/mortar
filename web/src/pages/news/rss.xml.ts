import rss from '@astrojs/rss'
import { posts } from '../../lib/news.ts'
import { SITE } from '../../lib/site.ts'

export async function GET() {
  return rss({
    title: 'Mortar news',
    description: "Mortar's release posts and how-tos.",
    site: SITE,
    items: (await posts()).map((p) => ({
      title: p.release ? `Mortar ${p.release.version}: ${p.title}` : p.title,
      description: p.summary,
      pubDate: p.date,
      link: `/news/${p.slug}/`,
      ...(p.release ? { content: p.release.html } : {}),
    })),
  })
}
