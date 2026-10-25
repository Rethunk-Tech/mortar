import { useLingui } from '@lingui/react/macro'
import { Box, LinearProgress, Typography } from '@mui/material'
import { useEffect } from 'react'
import { ImportsModsFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { MOD_ORGANIZER_2, STARDROP, VORTEX } from '../brand/productNames.ts'
import { localId } from '../mods/dependents.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useGameModsDialog } from '../profiles/gameModsDialog.ts'
import { useProfiles } from '../profiles/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { AppDialog } from '../shell/dialog/AppDialog.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
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
      <Box
        component="ul"
        sx={{ m: 0, pl: space.pad, fontSize: 'base.fontSize', color: 'text.secondary' }}
      >
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
  const { data: modsFolder } = useLoaded(
    () => ImportsModsFolder(game),
    [game],
    false,
    reportUnexpected,
  )
  if (flow.tab === 'choose') {
    return (
      <ImportSources
        modsFolder={modsFolder}
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

  const { preview } = flow
  const hasMods = preview !== null && preview.mods.length > 0
  const steps: Record<TabId, string> = {
    choose: '',
    link: t`Link or code`,
    file: t`File`,
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
  const canBack = preview !== null || (flow.tab !== 'choose' && !flow.external)
  return (
    <AppDialog
      open={true}
      onClose={flow.busy ? () => undefined : flow.dismiss}
      onBack={canBack && !flow.busy ? back : undefined}
      title={manager ? t`Import from ${manager}` : t`Import a profile`}
      subtitle={step}
      // A link or file to choose needs a narrow dialog; the preview's tiles need the room.
      size={preview ? 'xl' : 'lg'}
      flush={true}
      footer={
        preview ? (
          <ImportFooter
            flow={flow}
            preview={preview}
            summary={summarize(preview.mods, flow.excluded)}
            targetName={targetName ?? ''}
            signedIn={signedIn}
          />
        ) : null
      }
    >
      {/* Reading a link, code or file, or importing, can take a while: a thin bar says it is under way. */}
      <Box sx={{ height: 2, flexShrink: 0 }}>
        {flow.busy ? <LinearProgress sx={{ height: 2 }} aria-label={t`Working`} /> : null}
      </Box>
      {preview ? (
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
      ) : (
        <ImportPane flow={flow} game={game} close={close} />
      )}
    </AppDialog>
  )
}

// SOURCE_NAMES are the product names of the mod managers Mortar imports profiles from, keyed by migrate's source kind.
const SOURCE_NAMES: Record<string, string> = {
  mo2: MOD_ORGANIZER_2,
  stardrop: STARDROP,
  vortex: VORTEX,
}

export function ImportDialog() {
  const request = useImportDialog((s) => s.request)
  return (
    <>
      <SaveImportedTemplateHost />
      {request ? <Body key={request.run} request={request} /> : null}
    </>
  )
}
