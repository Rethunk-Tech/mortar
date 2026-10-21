import { useLingui } from '@lingui/react/macro'
import { Box, Dialog, IconButton, Typography } from '@mui/material'
import { ArrowLeft, X } from 'lucide-react'
import { useEffect } from 'react'
import { localId } from '../mods/dependents.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useGameModsDialog } from '../profiles/gameModsDialog.ts'
import { useProfiles } from '../profiles/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { Tooltip } from '../shell/Tooltip.tsx'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { CollectionNotes } from './CollectionNotes.tsx'
import { CompareSummary } from './CompareSummary.tsx'
import { ImportFooter } from './ImportFooter.tsx'
import { ImportInput } from './ImportInput.tsx'
import { Tiles } from './ImportPreview.tsx'
import { type ImportSource, ImportSources } from './ImportSources.tsx'
import { missingModName, summarize } from './logic.ts'
import { ManagerPane } from './ManagerPane.tsx'
import { SaveImportedTemplateHost } from './SaveImportedTemplate.tsx'
import { seedPreview } from './seedPreview.ts'
import { type ImportRequest, useImportDialog } from './store.ts'
import { type Tab as TabId, useImportFlow } from './useImportFlow.ts'

const dialogSx = {
  bgcolor: 'var(--mortar-panel-solid)',
  border: '1px solid var(--mortar-hairline)',
  maxWidth: 'none',
  maxHeight: 'none',
  overflow: 'hidden',
  borderRadius: '6px',
}

function MissingMods({
  ids,
  mods,
}: {
  ids: string[]
  mods: { name: string; ids?: string[] | null }[]
}) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        mb: 1,
        px: space.pad,
        py: space.gap,
        bgcolor: calloutFill('warning'),
        border: '1px solid',
        borderColor: calloutLine('warning'),
        borderRadius: '4px',
      }}
    >
      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
        {t`Not found in the Mods folder`}
      </Typography>
      <Box component="ul" sx={{ m: 0, pl: space.pad, fontSize: 13, color: 'text.secondary' }}>
        {ids.map((id) => {
          const name = missingModName(id, mods)
          return (
            <li key={id} title={localId(id)}>
              {name ?? localId(id)}
            </li>
          )
        })}
      </Box>
    </Box>
  )
}

function useSeedPreview(
  request: ImportRequest,
  flow: {
    setTab: (tab: TabId) => void
    setText: (text: string) => void
    previewLink: (value: string) => Promise<void>
    previewFile: (file: string) => Promise<void>
    previewData: (data: string) => Promise<void>
    previewExternal: (value: NonNullable<ImportRequest['external']>) => Promise<void>
    previewCollectionUpdate: () => Promise<void>
  },
) {
  const {
    setTab,
    setText,
    previewLink,
    previewFile,
    previewData,
    previewExternal,
    previewCollectionUpdate,
  } = flow
  useEffect(() => {
    seedPreview(request, {
      setTab,
      setText,
      previewLink,
      previewFile,
      previewData,
      previewExternal,
      previewCollectionUpdate,
    }).catch(reportUnexpected)
  }, [
    request,
    setTab,
    setText,
    previewLink,
    previewFile,
    previewData,
    previewExternal,
    previewCollectionUpdate,
  ])
}

// Before a preview: the grid of places a profile comes from, the link or file field, or other managers' profiles.
function ImportPane({
  flow,
  game,
  close,
}: {
  flow: ReturnType<typeof useImportFlow>
  game: string
  close: () => void
}) {
  if (flow.tab === 'choose') {
    return (
      <ImportSources
        onChoose={(picked: ImportSource) => {
          if (picked === 'folder') {
            close()
            useGameModsDialog.getState().open(game)
            return
          }
          flow.setTab(picked)
        }}
      />
    )
  }
  if (flow.tab === 'manager') {
    return (
      <ManagerPane
        game={game}
        onFolder={(folder) => flow.previewLocalProfile(folder).catch(reportUnexpected)}
        onExternal={(found) => flow.previewExternal(found).catch(reportUnexpected)}
        onDone={close}
      />
    )
  }
  return <ImportInput flow={flow} />
}

function Body({ request }: { request: ImportRequest }) {
  const { t } = useLingui()
  const close = useImportDialog((s) => s.close)
  const game = useCurrentGame()
  const targetName = useProfiles((s) => s.profiles.find((p) => p.id === request.profileId)?.name)
  const signedIn = useNexus((s) => s.signedIn)
  const flow = useImportFlow(game, request.profileId, close, request.external)
  const { setTab } = flow
  const source = flow.external?.source ?? ''
  const manager = SOURCE_NAMES[source] ?? source
  useSeedPreview(request, flow)

  useEffect(() => {
    useImportDialog.setState({ busy: flow.busy })
  }, [flow.busy])

  const { preview } = flow
  const hasMods = preview !== null && preview.mods.length > 0
  const steps: Record<TabId, string> = {
    choose: '',
    link: t`A link or code`,
    file: t`A file`,
    manager: t`Another mod manager`,
  }
  const step = flow.external ? '' : steps[flow.tab]
  const back = () => {
    const previewing = preview !== null
    const typed = flow.text
    flow.reset()
    if (previewing) {
      // Back from a preview returns to what was pasted, so it can be corrected.
      flow.setText(typed)
    } else {
      setTab('choose')
    }
  }
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
        maxHeight: 'calc(100vh - 48px)',
        // A link or file to choose needs a narrow dialog; the preview's tiles need the room.
        width: preview ? 'min(1180px, calc(100vw - 72px))' : 'min(720px, calc(100vw - 72px))',
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'stretch',
          height: 46,
          flexShrink: 0,
          borderBottom: '1px solid var(--mortar-hairline)',
        }}
      >
        {preview || (flow.tab !== 'choose' && !flow.external) ? (
          <Box sx={{ display: 'flex', alignItems: 'center', pl: '8px' }}>
            <TipIconButton label={t`Back`} disabled={flow.busy} onClick={back}>
              <ArrowLeft size={18} />
            </TipIconButton>
          </Box>
        ) : null}
        <Typography
          component="h2"
          sx={{
            display: 'flex',
            alignItems: 'center',
            px: '18px',
            flexGrow: 1,
            minWidth: 0,
            fontSize: 17,
            whiteSpace: 'nowrap',
          }}
        >
          {manager ? t`Import from ${manager}` : t`Import a profile`}
          {step ? (
            <Box component="span" sx={{ ml: 1.25, color: 'text.secondary', fontSize: 14 }}>
              {step}
            </Box>
          ) : null}
        </Typography>
        <Tooltip title={t`Close`} describeChild={true}>
          <span>
            <IconButton
              aria-label={t`Close`}
              onClick={flow.dismiss}
              disabled={flow.busy}
              sx={{ width: 50, borderRadius: 0, flexShrink: 0 }}
            >
              <X size={16} />
            </IconButton>
          </span>
        </Tooltip>
      </Box>
      {preview ? (
        <>
          <Box
            sx={{
              flex: '1 1 auto',
              minHeight: 0,
              overflowY: 'auto',
              p: space.gap,
              bgcolor: 'var(--mortar-overlay-20)',
            }}
          >
            {flow.external?.missing && flow.external.missing.length > 0 ? (
              <MissingMods ids={flow.external.missing} mods={preview.mods} />
            ) : null}
            {targetName && !flow.external ? (
              <CompareSummary preview={preview} targetName={targetName} />
            ) : null}
            {preview.collection ? <CollectionNotes collection={preview.collection} /> : null}
            {hasMods ? (
              <Tiles mods={preview.mods} excluded={flow.excluded} onToggle={flow.toggle} />
            ) : (
              <Typography sx={{ p: space.pad, textAlign: 'center', color: 'text.secondary' }}>
                {t`This profile has no mods to download`}
              </Typography>
            )}
          </Box>
          <ImportFooter
            flow={flow}
            preview={preview}
            summary={summarize(preview.mods, flow.excluded)}
            targetName={targetName ?? ''}
            signedIn={signedIn}
          />
        </>
      ) : (
        <ImportPane flow={flow} game={game} close={close} />
      )}
    </Box>
  )
}

// SOURCE_NAMES are the product names of the mod managers Mortar imports profiles from, keyed by migrate's source kind.
const SOURCE_NAMES: Record<string, string> = {
  mo2: 'Mod Organizer 2',
  stardrop: 'Stardrop',
  vortex: 'Vortex',
}

export function ImportDialog() {
  const { t } = useLingui()
  const request = useImportDialog((s) => s.request)
  const dismiss = useImportDialog((s) => s.close)
  const busy = useImportDialog((s) => s.busy)
  return (
    <>
      <SaveImportedTemplateHost />
      <Dialog
        open={request !== null}
        onClose={busy ? undefined : dismiss}
        maxWidth={false}
        slotProps={{ paper: { sx: dialogSx, 'aria-label': t`Import a profile` } }}
      >
        {request ? <Body key={request.run} request={request} /> : null}
      </Dialog>
    </>
  )
}
