// The visitor's system as the browser reports it, or null for one Mortar has no build for (macOS, phones).
export function visitorOS(): 'windows' | 'linux' | null {
  const ua = navigator.userAgent
  if (/Android|iPhone|iPad/i.test(ua)) {
    return null
  }
  if (/Windows/i.test(ua)) {
    return 'windows'
  }
  return /Linux|X11/i.test(ua) ? 'linux' : null
}
