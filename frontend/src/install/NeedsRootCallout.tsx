import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { Skip } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { Callout } from '../queue/Callout.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useInstall } from './store.ts'

export function NeedsRootCallout({ item }: { item: Item }) {
  const { t } = useLingui()
  const openRemap = useInstall((s) => s.openRemap)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === item.profileId))
  const ask = item.remap
  if (!ask) {
    return null
  }
  const profileName = profile?.name ?? ''
  const variants = (ask.variants ?? []).length > 0
  const { overlay } = ask
  let label = variants ? t`Choose a variant` : t`Choose a folder`
  let button = variants ? t`Choose variant…` : t`Choose folder…`
  let text = variants
    ? t`This archive holds several variants of the same mod. Pick the one to install; updates keep it.`
    : t`This archive has no SMAPI mod where Mortar expects one. Pick the folder that holds manifest.json.`
  if (overlay) {
    label = t`Place the optional file`
    button = t`Place…`
    text = t`This optional file replaces files of ${overlay.baseLabel}. Pick where its files go.`
  }
  return (
    <Callout
      item={item}
      label={label}
      text={text}
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
        onClick={() =>
          openRemap({
            game: item.game,
            profileId: item.profileId,
            profileName,
            key: ask.key,
            source: ask.source,
            ask,
            queueId: item.id,
          })
        }
        sx={{ alignSelf: 'flex-start' }}
      >
        {button}
      </Button>
    </Callout>
  )
}
