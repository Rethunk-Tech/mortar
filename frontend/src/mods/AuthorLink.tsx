import { Link } from '@mui/material'
import type { MouseEvent } from 'react'
import { useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { AuthorDialog } from './AuthorDialog.tsx'
import { splitManifestAuthors } from './authorNormalize.ts'

const linkSx = { fontSize: 'inherit', verticalAlign: 'baseline', textAlign: 'left' } as const
const emptyAuthor = '—'

export function AuthorLink({
  authorField,
  mod,
  profile,
}: {
  authorField: string
  mod?: Mod | undefined
  profile?: Profile | undefined
}) {
  const [open, setOpen] = useState(false)
  const [picked, setPicked] = useState('')
  const parts = splitManifestAuthors(authorField)
  const openAuthor = (name: string, e: MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setPicked(name)
    setOpen(true)
  }
  if (parts.length === 0) {
    if (authorField.trim() === '') {
      return <>{emptyAuthor}</>
    }
    return <>{authorField}</>
  }
  return (
    <>
      {parts.map((name, index) => (
        <span key={name}>
          {index > 0 ? ' & ' : null}
          <Link component="button" onClick={(e) => openAuthor(name, e)} sx={linkSx}>
            {name}
          </Link>
        </span>
      ))}
      <AuthorDialog
        open={open}
        author={picked}
        seedMod={mod}
        seedProfile={profile}
        onClose={() => setOpen(false)}
      />
    </>
  )
}
