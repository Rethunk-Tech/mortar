import { useLingui } from '@lingui/react/macro'
import { Box, Dialog, IconButton, Tab, Tabs, Tooltip, Typography } from '@mui/material'
import { X } from 'lucide-react'
import { useEffect } from 'react'
import { PreviewData } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import { paper } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { CompareSummary } from './CompareSummary.tsx'
import { ImportFooter } from './ImportFooter.tsx'
import { ImportInput } from './ImportInput.tsx'
import { Tiles } from './ImportPreview.tsx'
import { missingModName, summarize } from './logic.ts'
import { type ImportRequest, useImportDialog } from './store.ts'
import { type Tab as TabId, useImportFlow } from './useImportFlow.ts'

const dialogSx = {
  ...paper.sx,
  bgcolor: 'rgb(34,34,40)',
  border: '1px solid rgba(255,255,255,0.1)',
  width: 'min(1180px, calc(100% - 72px))',
  maxHeight: 'none',
  overflow: 'hidden',
  borderRadius: '6px',
}

function MissingMods({
  ids,
  mods,
}: {
  ids: string[]
  mods: { name: string; uniqueIds?: string[] | null }[]
}) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        mb: 1,
        px: 1.5,
        py: 1,
        bgcolor: 'rgba(243,180,22,0.12)',
        border: '1px solid rgba(243,180,22,0.35)',
        borderRadius: '4px',
      }}
    >
      <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
        {t`Not found in the Mods folder`}
      </Typography>
      <Box component="ul" sx={{ m: 0, pl: 2.5, fontSize: 13, color: 'text.secondary' }}>
        {ids.map((id) => {
          const name = missingModName(id, mods)
          return (
            <li key={id} title={id}>
              {name ?? id}
            </li>
          )
        })}
      </Box>
    </Box>
  )
}

function Body({ request }: { request: ImportRequest }) {
  const { t } = useLingui()
  const close = useImportDialog((s) => s.close)
  const game = useProfiles((s) => s.game?.id ?? 'stardew')
  const targetName = useProfiles((s) => s.profiles.find((p) => p.id === request.profileId)?.name)
  const signedIn = useNexus((s) => s.signedIn)
  const flow = useImportFlow(game, request.profileId, close, request.external)
  const { setTab, setText, previewLink, previewFile, previewExternal } = flow

  useEffect(() => {
    setTab(request.tab === 'link' ? 'link' : 'file')
    if (request.external) {
      previewExternal(request.external).catch(reportUnexpected)
    } else if (request.seed && request.tab === 'link') {
      setText(request.seed)
      previewLink(request.seed).catch(reportUnexpected)
    } else if (request.seed && request.tab === 'data') {
      PreviewData(game, request.seed, request.profileId).catch(reportUnexpected)
    } else if (request.seed) {
      previewFile(request.seed).catch(reportUnexpected)
    }
  }, [request, setTab, setText, previewLink, previewFile, previewExternal, game])

  useEffect(() => {
    useImportDialog.setState({ busy: flow.busy })
  }, [flow.busy])

  const { preview } = flow
  const hasMods = preview !== null && preview.mods.length > 0
  return (
    <Box
      role="dialog"
      aria-label={t`Import profile`}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
        maxHeight: 'calc(100vh - 48px)',
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'stretch',
          height: 46,
          flexShrink: 0,
          borderBottom: '1px solid rgba(255,255,255,0.1)',
        }}
      >
        <Typography
          component="h2"
          sx={{
            display: 'flex',
            alignItems: 'center',
            px: '18px',
            minWidth: 200,
            fontSize: 17,
            whiteSpace: 'nowrap',
          }}
        >
          {flow.external
            ? t`Import from ${SOURCE_NAMES[flow.external.source] ?? flow.external.source}`
            : t`Import profile from…`}
        </Typography>
        <Tabs
          value={flow.tab}
          onChange={(_, next: TabId) => {
            flow.reset()
            setTab(next)
          }}
          aria-label={t`Where the profile comes from`}
          sx={{
            flexGrow: 1,
            minHeight: 46,
            // Another mod manager's profile has no link or file to switch to.
            visibility: flow.external ? 'hidden' : 'visible',
            '& .MuiTab-root': {
              flex: 1,
              maxWidth: 'none',
              minHeight: 46,
              fontSize: 13,
              fontWeight: 600,
              color: 'text.secondary',
              '&.Mui-selected': { color: 'primary.main', fontWeight: 700 },
            },
          }}
        >
          <Tab value="link" label={t`Link`} disabled={flow.busy} />
          <Tab value="file" label={t`.mortar file`} disabled={flow.busy} />
        </Tabs>
        <Tooltip title={t`Close`}>
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
              p: 1,
              bgcolor: 'rgba(0,0,0,0.2)',
            }}
          >
            {flow.external?.missing && flow.external.missing.length > 0 ? (
              <MissingMods ids={flow.external.missing} mods={preview.mods} />
            ) : null}
            {targetName && !flow.external ? (
              <CompareSummary preview={preview} targetName={targetName} />
            ) : null}
            {hasMods ? (
              <Tiles mods={preview.mods} excluded={flow.excluded} onToggle={flow.toggle} />
            ) : (
              <Typography sx={{ p: 3, textAlign: 'center', color: 'text.secondary' }}>
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
        <ImportInput flow={flow} />
      )}
    </Box>
  )
}

// SOURCE_NAMES are the product names of the mod managers Mortar imports profiles from, keyed by migrate's source kind.
const SOURCE_NAMES: Record<string, string> = { stardrop: 'Stardrop', vortex: 'Vortex' }

export function ImportDialog() {
  const request = useImportDialog((s) => s.request)
  const dismiss = useImportDialog((s) => s.close)
  const busy = useImportDialog((s) => s.busy)
  return (
    <Dialog
      open={request !== null}
      onClose={busy ? undefined : dismiss}
      maxWidth={false}
      slotProps={{ paper: { ...paper, sx: dialogSx } }}
    >
      {request ? <Body key={request.run} request={request} /> : null}
    </Dialog>
  )
}
