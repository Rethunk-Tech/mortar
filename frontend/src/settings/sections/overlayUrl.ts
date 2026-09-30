export function overlayFileUrl(pagePath: string, port: number, token: string): string {
  const path = pagePath.replaceAll('\\', '/')
  const href = path.startsWith('/') ? `file://${path}` : `file:///${path}`
  const url = new URL(href)
  url.searchParams.set('port', String(port))
  url.searchParams.set('token', token)
  return url.toString()
}
