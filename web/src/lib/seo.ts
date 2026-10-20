// Structured data for search engines and the address of each page's link-preview picture.
import { appVersion } from './faq.ts'
import { SITE } from './site.ts'

const EDGE_SLASHES = /^\/|\/$/g
const HTML_EXT = /\.html$/

export interface Crumb {
  name: string
  href: string
}

export function appLd(): object {
  return {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Mortar',
    applicationCategory: 'GameApplication',
    operatingSystem: 'Windows, Linux',
    softwareVersion: appVersion(),
    url: `${SITE}/`,
    downloadUrl: `${SITE}/download/`,
    license: 'https://www.gnu.org/licenses/agpl-3.0.html',
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  }
}

export function breadcrumbLd(crumbs: Crumb[]): object {
  return {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [{ name: 'Mortar', href: '/' }, ...crumbs].map((c, i) => ({
      '@type': 'ListItem',
      position: i + 1,
      name: c.name,
      item: `${SITE}${c.href}`,
    })),
  }
}

export function faqLd(qs: { q: string; text: string }[]): object {
  return {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: qs.map((x) => ({
      '@type': 'Question',
      name: x.q,
      acceptedAnswer: { '@type': 'Answer', text: x.text },
    })),
  }
}

/** The link-preview picture the build writes for the page at `path`: /og/<path>.png, the home page as /og/index.png. */
export function ogPath(path: string): string {
  const slug = path.replace(EDGE_SLASHES, '').replace(HTML_EXT, '')
  return `/og/${slug || 'index'}.png`
}
