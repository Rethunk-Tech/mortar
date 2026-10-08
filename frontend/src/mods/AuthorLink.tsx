import { Link } from '@mui/material'
import type { MouseEvent } from 'react'
import { useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { AuthorDialog } from './AuthorDialog.tsx'
import { authorListParts, splitManifestAuthors } from './authorNormalize.ts'

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
  // A joiner sits after the name before it, so that name keys it.
  let lastName = ''
  const listParts = authorListParts(authorField).map((part) => {
    if (part.type === 'element') {
      lastName = part.value
    }
    return { ...part, key: part.type === 'element' ? part.value : `after-${lastName}` }
  })
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
      {listParts.map((part) =>
        part.type === 'literal' ? (
          <span key={part.key}>{part.value}</span>
        ) : (
          <Link
            key={part.key}
            component="button"
            onClick={(e) => openAuthor(part.value, e)}
            sx={linkSx}
          >
            {part.value}
          </Link>
        ),
      )}
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
