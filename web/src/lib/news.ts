// /news/: a post per release, its date and notes from the release itself, and how-to posts. A release with no post
// written yet still gets one, titled by its version, so cutting a release never waits on the site.
import { type CollectionEntry, getCollection } from 'astro:content'
import { type Release, releases } from './changelog.ts'

export interface Post {
  slug: string
  kind: 'release' | 'howto'
  title: string
  summary: string
  date: Date
  image?: string
  highlights: { area: string; title: string; text: string; image: string }[]
  release?: Release
  entry?: CollectionEntry<'news'>
}

const slugOf = (version: string) => `mortar-${version.replaceAll('.', '-')}`
const FIRST_BULLET = /^- (.+)$/m

export async function posts(): Promise<Post[]> {
  const entries = await getCollection('news')
  const byVersion = new Map(entries.flatMap((e) => (e.data.version ? [[e.data.version, e]] : [])))
  const fromReleases = releases().map((r): Post => {
    const e = byVersion.get(r.version)
    const first = FIRST_BULLET.exec(r.md)?.[1] ?? ''
    return {
      slug: e?.id ?? slugOf(r.version),
      kind: 'release',
      title: e?.data.title ?? `Mortar ${r.version}`,
      summary: e?.data.summary ?? first,
      date: new Date(`${r.date}T00:00:00Z`),
      highlights: e?.data.highlights ?? [],
      release: r,
      ...(e ? { entry: e } : {}),
    }
  })
  const howtos = entries
    .filter((e) => e.data.kind === 'howto')
    .map((e): Post => {
      if (!e.data.date) {
        throw new Error(`site: news/${e.id}.md is a how-to and needs a date`)
      }
      return {
        slug: e.id,
        kind: 'howto',
        title: e.data.title,
        summary: e.data.summary,
        date: e.data.date,
        highlights: [],
        ...(e.data.image ? { image: e.data.image } : {}),
        entry: e,
      }
    })
  return [...fromReleases, ...howtos].sort((a, b) => b.date.getTime() - a.date.getTime())
}

export const longDate = (d: Date) =>
  d.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric', timeZone: 'UTC' })

export const shortDate = (d: Date) =>
  d.toLocaleDateString('en-US', { month: 'long', day: 'numeric', timeZone: 'UTC' })
