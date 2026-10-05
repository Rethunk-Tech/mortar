import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import {
  AnswerMerge,
  Skip,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { Callout } from '../queue/Callout.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'

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
    >
      {t`Install together`}
    </Button>
  )
  const separate = (
    <Button
      variant={addFirst ? 'outlined' : 'contained'}
      disabled={pending}
      onClick={() => run(() => AnswerMerge(item.id, false))}
    >
      {t`Install separately`}
    </Button>
  )
  return (
    <Callout
      item={item}
      label={t`Another file from a mod you have`}
      text={t`This file is from the same Nexus page as ${ask.label}. Install it with that mod to update them as one, or as a separate mod.`}
      actions={null}
    >
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', flexWrap: 'wrap' }}>
        <Button
          variant="outlined"
          color="inherit"
          onClick={() => Skip(item.id).catch(reportUnexpected)}
        >
          {t`Skip`}
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
    </Callout>
  )
}
