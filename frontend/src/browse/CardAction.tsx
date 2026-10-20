import { useLingui } from '@lingui/react/macro'
import { Button, Chip } from '@mui/material'
import { Download, ExternalLink, Plus } from 'lucide-react'
import { openPage } from '../mods/menu.ts'
import { useNav } from '../nav/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { OfflineGate } from '../shell/OfflineGate.tsx'
import { useOfflineReason } from '../shell/offlineText.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { NEXUS } from './browseConstants.ts'
import type { BrowseItem } from './browseTypes.ts'
import { CardProgress } from './CardProgress.tsx'
import type { CardState } from './cardState.ts'

// Premium downloads directly; free accounts use the files page's Mod Manager Download button, which Mortar picks up.
function NexusAction({
  premium,
  pending,
  onDownload,
  onOpenFiles,
}: {
  premium: boolean
  pending: boolean
  onDownload: () => void
  onOpenFiles: () => void
}) {
  const { t } = useLingui()
  const signedIn = useNexus((state) => state.signedIn)
  if (!signedIn) {
    const why = t`Sign in to download`
    return (
      <Tooltip title={why}>
        <Button
          size="small"
          variant="outlined"
          aria-label={why}
          onClick={() => useNav.getState().openSettings('accounts')}
        >
          {t`Sign in`}
        </Button>
      </Tooltip>
    )
  }
  if (premium) {
    return (
      <Button
        size="small"
        variant="contained"
        startIcon={<Download size={14} />}
        disabled={pending}
        onClick={onDownload}
      >
        {t`Download`}
      </Button>
    )
  }
  // The short label fits under a card's picture; the files page is where Mod Manager Download starts it.
  const why = t`Open files page`
  return (
    <Tooltip title={why}>
      <Button
        size="small"
        variant="contained"
        aria-label={why}
        startIcon={<ExternalLink size={14} />}
        onClick={onOpenFiles}
      >
        {t`Get`}
      </Button>
    </Tooltip>
  )
}

function AddButton({ pending, onAdd }: { pending: boolean; onAdd: () => void }) {
  const { t } = useLingui()
  return (
    <Button
      size="small"
      variant="contained"
      startIcon={<Plus size={14} />}
      disabled={pending}
      onClick={onAdd}
    >
      {t`Add`}
    </Button>
  )
}

interface CardActionProps {
  item: BrowseItem
  source: string
  state: CardState
  installed: boolean
  premium: boolean
  pending: boolean
  onAdd: () => void
  onDownload: () => void
  onOpenFiles: () => void
}

// The one control a card offers: progress, a status chip, or the source's own add or download button.
function sourceAction(props: CardActionProps) {
  const { source, premium, pending, onAdd, onDownload, onOpenFiles } = props
  if (source === NEXUS) {
    return (
      <NexusAction
        premium={premium}
        pending={pending}
        onDownload={onDownload}
        onOpenFiles={onOpenFiles}
      />
    )
  }
  return <AddButton pending={pending} onAdd={onAdd} />
}

function CardAction(props: CardActionProps) {
  const { t } = useLingui()
  const { item, source, state, installed } = props
  const offline = useOfflineReason([source])
  if (item.bundled) {
    return <Chip size="small" label={t`Built in`} title={t`Installed by Mortar`} />
  }
  if (state.kind !== 'idle') {
    return <CardProgress state={state} />
  }
  if (installed) {
    return <Chip size="small" label={t`Added`} title={t`In this profile`} />
  }
  if (item.external === true) {
    return (
      <Button size="small" variant="outlined" onClick={() => openPage(item.url)}>
        {t`Open page`}
      </Button>
    )
  }
  const action = sourceAction(props)
  return offline === '' ? action : <OfflineGate reason={offline}>{action}</OfflineGate>
}

export { CardAction }
