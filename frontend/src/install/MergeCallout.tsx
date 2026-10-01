import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import {
  AnswerMerge,
  Skip,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { accent } from '../mods/paper.ts'
import { LetterTile } from '../mods/parts.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'

const tile = (i: Item) => ({
  uniqueId: i.repo || String(i.modId),
  name: i.name || i.repo || String(i.modId),
  picture: '',
})

export function MergeCallout({ item }: { item: Item }) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  const ask = item.merge
  if (!ask) {
    return null
  }
  const addFirst = ask.defaultAdd
  const add = (
    <Button
      variant={addFirst ? 'contained' : 'outlined'}
      disabled={pending}
      onClick={() => run(() => AnswerMerge(item.id, true))}
      sx={{ whiteSpace: 'nowrap' }}
    >
      {t`Install together`}
    </Button>
  )
  const separate = (
    <Button
      variant={addFirst ? 'outlined' : 'contained'}
      disabled={pending}
      onClick={() => run(() => AnswerMerge(item.id, false))}
      sx={{ whiteSpace: 'nowrap' }}
    >
      {t`Install separately`}
    </Button>
  )
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        p: '14px',
        bgcolor: accent.fill,
        border: '1px solid',
        borderColor: 'primary.main',
        borderRadius: '8px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <LetterTile mod={tile(item)} size={44} />
        <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          <Typography sx={{ fontSize: 12, fontWeight: 700, letterSpacing: '0.04em' }}>
            {t`Another file from a mod you have`}
          </Typography>
          <Typography noWrap={true} sx={{ fontSize: 15, fontWeight: 600 }}>
            {item.fileName || item.name}
          </Typography>
        </Box>
      </Box>
      <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
        {t`This file comes from the same Nexus page as ${ask.label}. Install it together with that mod so they update as one, or as a separate mod.`}
      </Typography>
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', flexWrap: 'wrap' }}>
        <Button
          variant="outlined"
          color="inherit"
          onClick={() => Skip(item.id).catch(reportUnexpected)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Don't install`}
        </Button>
        {addFirst ? (
          <>
            {separate}
            {add}
          </>
        ) : (
          <>
            {add}
            {separate}
          </>
        )}
      </Box>
    </Box>
  )
}
