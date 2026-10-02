import { useLingui } from '@lingui/react/macro'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  IconButton,
  Tooltip,
  Typography,
} from '@mui/material'
import { ChevronDown, CircleCheck, CircleX, FolderPlus, RefreshCw, X } from 'lucide-react'
import { useState } from 'react'
import type { StoreApp } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import {
  AddLauncherRoot,
  RemoveLauncherRoot,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { LauncherLogo } from '../brand/launchers/LauncherLogo.tsx'
import { errorMessage } from '../toasts/report.ts'

const STATUS_ICON = 30
// Two columns once each can hold a row comfortably.
const COLUMN_MIN = 460

const pathSx = { fontSize: 13, userSelect: 'text', overflowWrap: 'anywhere' } as const

function LauncherRow({ launcher, refresh }: { launcher: StoreApp; refresh: () => void }) {
  const { t } = useLingui()
  const [error, setError] = useState('')
  const games = (launcher.games ?? []).map((g) => g.name)
  const roots = launcher.roots ?? []
  const custom = launcher.custom ?? []
  const looked = launcher.looked ?? []
  const run = (p: Promise<void>) =>
    p.then(
      () => {
        setError('')
        refresh()
      },
      (e: unknown) => setError(errorMessage(e)),
    )
  const add = async () => {
    try {
      const picked = await PickFolder(t`Choose a ${launcher.name} folder`)
      if (picked) {
        await run(AddLauncherRoot(launcher.id, picked))
      }
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  let summary = t`Not found`
  if (launcher.found) {
    summary = games.length > 0 ? games.join(', ') : t`Found, with no supported games yet`
  }
  return (
    <Accordion
      disableGutters={true}
      sx={{
        bgcolor: 'rgba(50,50,60,0.78)',
        backgroundImage: 'none',
        '&::before': { display: 'none' },
      }}
    >
      <AccordionSummary
        expandIcon={<ChevronDown size={18} />}
        sx={{ '& .MuiAccordionSummary-content': { alignItems: 'center', gap: 2, my: 1.25 } }}
      >
        <Box sx={{ display: 'flex', flexShrink: 0 }}>
          <LauncherLogo id={launcher.id} size={STATUS_ICON} />
        </Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography sx={{ fontSize: 17, fontWeight: 700 }}>{launcher.name}</Typography>
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }} noWrap={true}>
            {summary}
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', mr: 2.5 }}>
          {launcher.found ? (
            <CircleCheck size={STATUS_ICON} color="#0CDF64" aria-label={t`Found`} />
          ) : (
            <CircleX size={STATUS_ICON} color="#8A909A" aria-label={t`Not found`} />
          )}
        </Box>
      </AccordionSummary>
      <AccordionDetails sx={{ display: 'flex', flexDirection: 'column', gap: 1.25, pt: 0 }}>
        {roots.length > 0 ? (
          <Box>
            <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{t`Found in:`}</Typography>
            {roots.map((dir) => (
              <Typography key={dir} sx={pathSx}>
                {dir}
              </Typography>
            ))}
          </Box>
        ) : (
          <Box>
            <Typography
              sx={{ fontSize: 13, color: 'text.secondary' }}
            >{t`Mortar looked in:`}</Typography>
            <Box component="ul" sx={{ m: 0, pl: 2.5, color: 'text.secondary', ...pathSx }}>
              {looked.map((dir) => (
                <li key={dir}>{dir}</li>
              ))}
            </Box>
          </Box>
        )}
        {custom.length > 0 ? (
          <Box>
            <Typography
              sx={{ fontSize: 13, color: 'text.secondary' }}
            >{t`Folders you added:`}</Typography>
            {custom.map((dir) => (
              <Box key={dir} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <Typography sx={{ ...pathSx, flex: 1 }}>{dir}</Typography>
                <Tooltip title={t`Stop searching this folder`}>
                  <IconButton
                    size="small"
                    aria-label={t`Remove ${dir}`}
                    onClick={() => run(RemoveLauncherRoot(launcher.id, dir))}
                  >
                    <X size={14} />
                  </IconButton>
                </Tooltip>
              </Box>
            ))}
          </Box>
        ) : null}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Button
            variant={launcher.found ? 'outlined' : 'contained'}
            color={launcher.found ? 'inherit' : 'primary'}
            size="small"
            startIcon={<FolderPlus size={15} />}
            onClick={add}
          >
            {t`Add folder…`}
          </Button>
          <Button
            variant="text"
            color="inherit"
            size="small"
            startIcon={<RefreshCw size={14} />}
            onClick={refresh}
          >
            {t`Rescan`}
          </Button>
        </Box>
        {error ? (
          <Typography role="alert" sx={{ fontSize: 13, color: 'error.light' }}>
            {error}
          </Typography>
        ) : null}
      </AccordionDetails>
    </Accordion>
  )
}

// The launchers Mortar reads, one row each: found or not, the games in it, and every folder it uses.
export function LauncherList({
  launchers,
  refresh,
}: {
  launchers: StoreApp[]
  refresh: () => void
}) {
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: `repeat(auto-fit, minmax(min(${COLUMN_MIN}px, 100%), 1fr))`,
        alignItems: 'start',
        gap: 1,
      }}
    >
      {launchers.map((l) => (
        <LauncherRow key={l.id} launcher={l} refresh={refresh} />
      ))}
    </Box>
  )
}
