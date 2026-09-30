export type PageHost = 'nexus' | 'github' | ''
export type ModAction = 'toggle' | 'details' | 'page' | 'files' | 'pin' | 'skip' | 'remove'

export interface MenuState {
  enabled: boolean
  host: PageHost
  pinned: boolean
  skipVersion: string
  hasUpdate: boolean
}

// Mod pages are on Nexus or GitHub only.
export const hostOf = (url: string | undefined): PageHost => {
  if (!url) {
    return ''
  }
  return new URL(url).hostname === 'github.com' ? 'github' : 'nexus'
}

// The actions of a mod's menu, in menu order; Remove is the last and is set apart by a divider.
export function modActions(s: MenuState): ModAction[] {
  const out: ModAction[] = ['toggle', 'details']
  if (s.host !== '') {
    out.push('page')
  }
  out.push('files')
  out.push('pin')
  if (s.hasUpdate || s.skipVersion !== '') {
    out.push('skip')
  }
  out.push('remove')
  return out
}
