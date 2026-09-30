export type PageHost = 'nexus' | 'github' | ''
export type ModAction = 'toggle' | 'details' | 'page' | 'files' | 'remove'

export interface MenuState {
  enabled: boolean
  host: PageHost
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
  out.push('remove')
  return out
}
