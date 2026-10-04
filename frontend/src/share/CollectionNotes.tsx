import { useLingui } from '@lingui/react/macro'
import { Box, Link, Typography } from '@mui/material'
import type { CollectionInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import { openPage } from '../mods/menu.ts'
import { type InstructionPart, parseInstructions } from './instructions.ts'

const panelSx = {
  mb: 1,
  px: 1.5,
  py: 1,
  bgcolor: 'var(--mortar-card-hover)',
  borderRadius: '4px',
}

function PageLink({ part }: { part: InstructionPart }) {
  const { url } = part
  return url === undefined ? (
    part.text
  ) : (
    <Link
      component="button"
      type="button"
      onClick={() => openPage(url)}
      sx={{ verticalAlign: 'baseline', textAlign: 'left' }}
    >
      {part.text}
    </Link>
  )
}

function Instructions({ text }: { text: string }) {
  const { t } = useLingui()
  const lines = parseInstructions(text)
  return (
    <Box sx={panelSx}>
      <Typography variant="body2" sx={{ fontWeight: 600, mb: 0.5 }}>
        {t`Curator's instructions`}
      </Typography>
      <Box sx={{ maxHeight: 160, overflowY: 'auto', fontSize: 13, color: 'text.secondary' }}>
        {lines.map(({ at, parts }) => (
          <Box key={at} sx={{ minHeight: '1.4em' }}>
            {parts.map((part) => (
              <PageLink key={part.at} part={part} />
            ))}
          </Box>
        ))}
      </Box>
    </Box>
  )
}

function Externals({ items }: { items: NonNullable<CollectionInfo['external']> }) {
  const { t } = useLingui()
  return (
    <Box sx={panelSx}>
      <Typography variant="body2" sx={{ fontWeight: 600 }}>
        {t`Install yourself`}
      </Typography>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', mb: 0.5 }}>
        {t`These come from outside Nexus, so Mortar does not download them.`}
      </Typography>
      <Box component="ul" sx={{ m: 0, pl: 2.5, fontSize: 13 }}>
        {items.map((r) => (
          <li key={`${r.name}:${r.url}`}>
            {r.url ? <PageLink part={{ at: 0, text: r.name, url: r.url }} /> : r.name}
            <Typography component="span" sx={{ fontSize: 12, color: 'text.secondary' }}>
              {[r.version, r.author].filter(Boolean).map((s) => ` · ${s}`)}
            </Typography>
          </li>
        ))}
      </Box>
    </Box>
  )
}

function DetailsNote({ details }: { details: string }) {
  const { t } = useLingui()
  const notes: Record<string, string> = {
    archive: t`Your Premium account also applies the curator's installer choices and config files.`,
    listed: t`Installers ask as usual; Premium accounts also get the curator's choices and configs.`,
  }
  const note = notes[details]
  return note ? (
    <Typography sx={{ mb: 1, fontSize: 12, color: 'text.secondary' }}>{note}</Typography>
  ) : null
}

/** What a Nexus collection shows besides its mods: the curator's text, what to install by hand, and what Premium adds. */
export function CollectionNotes({ collection }: { collection: CollectionInfo }) {
  const external = collection.external ?? []
  return (
    <>
      {collection.instructions.trim() ? <Instructions text={collection.instructions} /> : null}
      {external.length > 0 ? <Externals items={external} /> : null}
      <DetailsNote details={collection.details} />
    </>
  )
}
