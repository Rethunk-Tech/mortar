// Resolves every download link on the site against the GitHub releases it points at, so a renamed or missing asset
// fails here instead of on a visitor's click. release.yml runs it once a release is published; locally it is
// `bun scripts/site-downloads.ts` (set GH_TOKEN to avoid the anonymous API rate limit).
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const SITE = 'site'
// download.js fills data-asset links from this repo's latest release, putting the version where VERSION stands.
const ASSET_REPO = 'Rethunk-Tech/mortar'
const LATEST_ASSET =
  /^https:\/\/github\.com\/([^/]+\/[^/]+)\/releases\/latest\/download\/([^/?#]+)$/
const PACKAGE_URL = /https:\/\/mortar\.rethunk\.tech\/packages\/[^\s"'<]+/g
const BREW = /brew install ([\w-]+)\/([\w-]+)\/([\w-]+)/g
const FORMULA_ASSET =
  /url "https:\/\/github\.com\/([^/]+\/[^/]+)\/releases\/download\/([^/]+)\/([^"]+)"\s+sha256 "([0-9a-f]{64})"/g

interface Release {
  tag_name: string
  assets: { name: string; digest?: string | null }[]
}

interface Anchor {
  page: string
  href?: string
  asset?: string
}

function attr(tag: string, name: string): string | undefined {
  return new RegExp(`\\s${name}="([^"]*)"`).exec(tag)?.[1]
}

function anchorsOf(page: string, html: string): Anchor[] {
  return [...html.matchAll(/<a\b[^>]*>/g)].map(([tag]) => ({
    page,
    href: attr(tag, 'href'),
    asset: attr(tag, 'data-asset'),
  }))
}

function latestAsset(a: Anchor): RegExpExecArray | null {
  return a.href ? LATEST_ASSET.exec(a.href) : null
}

/** The repos whose latest release the anchors need. */
function reposOf(anchors: Anchor[]): string[] {
  const repos = new Set<string>()
  for (const a of anchors) {
    const m = latestAsset(a)
    if (m) {
      repos.add(m[1])
    }
    if (a.asset) {
      repos.add(ASSET_REPO)
    }
  }
  return [...repos]
}

/** Every anchor whose asset is missing from the latest release it names, or whose href and data-asset disagree. */
function anchorProblems(anchors: Anchor[], latest: Record<string, Release>): string[] {
  const out: string[] = []
  const has = (repo: string, name: string) =>
    latest[repo]?.assets.some((x) => x.name === name) ?? false
  const tag = latest[ASSET_REPO]?.tag_name ?? ''
  const version = tag.replace(/^v/, '')
  for (const a of anchors) {
    const m = latestAsset(a)
    if (m && !has(m[1], m[2])) {
      out.push(
        `${a.page}: ${a.href} names no asset of ${m[1]} ${latest[m[1]]?.tag_name ?? '(no release)'}`,
      )
    }
    const name = a.asset?.replaceAll('VERSION', version)
    if (name && !has(ASSET_REPO, name)) {
      out.push(`${a.page}: data-asset ${a.asset} is ${name}, which ${ASSET_REPO} ${tag} lacks`)
    }
    if (name && m && m[2] !== name) {
      out.push(`${a.page}: href asset ${m[2]} differs from data-asset ${name}`)
    }
  }
  return out
}

function pages(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const path = join(dir, e.name)
    if (e.isDirectory()) {
      return e.name === 'vendor' ? [] : pages(path)
    }
    return e.name.endsWith('.html') ? [path] : []
  })
}

const token = process.env.GH_TOKEN ?? process.env.GITHUB_TOKEN
const apiHeaders: Record<string, string> = { Accept: 'application/vnd.github+json' }
if (token) {
  apiHeaders.Authorization = `Bearer ${token}`
}

async function release(repo: string, which: string): Promise<Release | undefined> {
  const res = await fetch(`https://api.github.com/repos/${repo}/releases/${which}`, {
    headers: apiHeaders,
  })
  return res.ok ? ((await res.json()) as Release) : undefined
}

async function packageProblems(text: string): Promise<string[]> {
  const out: string[] = []
  for (const url of new Set(text.match(PACKAGE_URL))) {
    const res = await fetch(url, { method: 'HEAD' })
    if (!res.ok) {
      out.push(`${url} answers ${res.status}`)
    }
  }
  return out
}

/** A Homebrew formula's pinned assets must exist with the digest it pins; the tap bumps itself every six hours, so a
 * formula one release behind is only a warning. */
async function formulaProblems(formula: string, name: string, latest: Record<string, Release>) {
  const out: string[] = []
  const pinned = [...formula.matchAll(FORMULA_ASSET)]
  if (pinned.length === 0) {
    out.push(`brew ${name}: the formula pins no release asset`)
  }
  for (const [, repo, tag, asset, sha] of pinned) {
    const found = (await release(repo, `tags/${tag}`))?.assets.find((x) => x.name === asset)
    if (!found) {
      out.push(`brew ${name}: ${repo} ${tag} has no asset ${asset}`)
    } else if (found.digest && found.digest !== `sha256:${sha}`) {
      out.push(`brew ${name}: ${asset} sha256 ${sha} differs from the release's ${found.digest}`)
    }
    if (latest[repo] && tag !== latest[repo].tag_name) {
      console.warn(`brew ${name} still pins ${tag}; the latest release is ${latest[repo].tag_name}`)
    }
  }
  return out
}

async function brewProblems(text: string, latest: Record<string, Release>): Promise<string[]> {
  const out: string[] = []
  for (const [, owner, tap, name] of text.matchAll(BREW)) {
    const url = `https://raw.githubusercontent.com/${owner}/homebrew-${tap}/HEAD/Formula/${name}.rb`
    const res = await fetch(url)
    if (res.ok) {
      out.push(...(await formulaProblems(await res.text(), name, latest)))
    } else {
      out.push(`brew ${owner}/${tap}/${name}: ${url} answers ${res.status}`)
    }
  }
  return out
}

async function main() {
  const html = Object.fromEntries(pages(SITE).map((p) => [p, readFileSync(p, 'utf8')]))
  const anchors = Object.entries(html).flatMap(([page, body]) => anchorsOf(page, body))
  const latest: Record<string, Release> = {}
  const problems: string[] = []
  for (const repo of reposOf(anchors)) {
    const rel = await release(repo, 'latest')
    if (rel) {
      latest[repo] = rel
    } else {
      problems.push(`${repo} has no latest release the API returns`)
    }
  }
  const all = Object.values(html).join('\n')
  problems.push(
    ...anchorProblems(anchors, latest),
    ...(await packageProblems(all)),
    ...(await brewProblems(all, latest)),
  )
  if (problems.length > 0) {
    console.error(problems.join('\n'))
    process.exit(1)
  }
  console.log(
    `site downloads: ${anchors.length} links on ${Object.keys(html).length} pages resolve`,
  )
}

if (import.meta.main) {
  await main()
}

export { anchorProblems, anchorsOf, reposOf }
