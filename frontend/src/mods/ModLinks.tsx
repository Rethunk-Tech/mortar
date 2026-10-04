import { Link } from '@mui/material'
import type { ReactNode } from 'react'
import { useTab } from '../game/tab.ts'
import { useDetail } from './detail.ts'
import type { ModLink } from './modLinks.ts'

const showInMods = ({ key, uniqueId }: ModLink) => {
  useTab.getState().setTab('mods')
  useDetail.getState().showAfterLoad({ key, uniqueId })
}

// LinkedText turns the first mention of each mod's name into a link that opens that mod on the Mods tab.
export function LinkedText({ text, links }: { text: string; links: ModLink[] }) {
  const parts: ReactNode[] = []
  let rest = text
  for (const link of links) {
    const at = link.name === '' ? -1 : rest.indexOf(link.name)
    if (at >= 0) {
      parts.push(rest.slice(0, at))
      parts.push(
        <Link
          key={`${link.key}/${link.uniqueId}`}
          component="button"
          color="inherit"
          onClick={() => showInMods(link)}
          sx={{ font: 'inherit', textAlign: 'left', verticalAlign: 'baseline' }}
        >
          {link.name}
        </Link>,
      )
      rest = rest.slice(at + link.name.length)
    }
  }
  parts.push(rest)
  return parts
}
