import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { FomodPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import { Skip } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { useFomod } from '../fomod/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { Callout } from './Callout.tsx'

function choose(item: Item) {
  FomodPreview(item.game, item.profileId, item.fomodKey ?? '', {}).then((ask) => {
    useFomod.getState().open({
      game: item.game,
      profileId: item.profileId,
      key: item.fomodKey ?? '',
      source: ask.source,
      queueId: item.id,
      ask,
    })
  }, reportUnexpected)
}

// Reopens the installer choices when the dialog was closed, or the app restarted with the download waiting.
export function QueueNeedsFomod({ items }: { items: Item[] }) {
  const { t } = useLingui()
  return (
    <>
      {items.map((item) => (
        <Callout
          key={item.id}
          item={item}
          label={t`Choose options`}
          text={t`This mod's installer has options to pick before it installs.`}
          actions={
            <Button
              variant="outlined"
              color="inherit"
              onClick={() => Skip(item.id).catch(reportUnexpected)}
            >
              {t`Skip`}
            </Button>
          }
        >
          <Button
            variant="contained"
            disabled={!item.fomodKey}
            onClick={() => choose(item)}
            sx={{ alignSelf: 'flex-start' }}
          >
            {t`Choose options…`}
          </Button>
        </Callout>
      ))}
    </>
  )
}
