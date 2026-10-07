import { useLingui } from '@lingui/react/macro'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  Typography,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { ChevronDown, CircleCheck, CircleX, Folder, FolderPlus, RefreshCw, X } from 'lucide-react'
import { useState } from 'react'
import type { StoreApp } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import {
  AddLauncherRoot,
  RemoveLauncherRoot,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { LauncherLogo } from '../brand/launchers/LauncherLogo.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'

const STATUS_ICON = 30
// Two columns once each can hold a row comfortably.
const COLUMN_MIN = 460

const pathSx = { fontSize: 13, userSelect: 'text', overflowWrap: 'anywhere' } as const

const MISSING_OPACITY = 0.5

function FolderRow({
  dir,
  note,
  found,
  onRemove,
}: {
  dir: string
  note: string
  found: boolean
  onRemove?: () => void
}) {
  const { t } = useLingui()
  return (
    <Box
      sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minHeight: 44, px: space.pad }}
    >
      <Folder
        size={16}
        aria-hidden={true}
        style={{ flexShrink: 0, opacity: found ? 1 : MISSING_OPACITY }}
      />
      <Typography
        title={dir}
        sx={{ ...pathSx, flex: 1, minWidth: 0, color: found ? 'text.primary' : 'text.secondary' }}
        noWrap={true}
      >
        {dir}
      </Typography>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'nowrap' }}>
        {note}
      </Typography>
      {onRemove ? (
        <TipIconButton label={t`Remove ${{ name: dir }}`} onClick={onRemove}>
          <X size={14} />
        </TipIconButton>
      ) : null}
    </Box>
  )
}

function LauncherRow({
  launcher,
  refresh,
  open,
}: {
  launcher: StoreApp
  refresh: () => void
  open?: boolean
}) {
  const { t } = useLingui()
  const { success, text } = useTheme().palette
  const ok = success.main
  const [error, setError] = useState<InlineError | null>(null)
  const games = (launcher.games ?? []).map((g) => g.name)
  const roots = launcher.roots ?? []
  const custom = launcher.custom ?? []
  const looked = launcher.looked ?? []
  const run = (p: Promise<void>) =>
    p.then(
      () => {
        setError(null)
        refresh()
      },
      (e: unknown) => setError(inlineError(e)),
    )
  const add = async () => {
    try {
      const picked = await PickFolder(t`Choose a ${launcher.name} folder`)
      if (picked) {
        await run(AddLauncherRoot(launcher.id, picked))
      }
    } catch (e) {
      setError(inlineError(e))
    }
  }
  let summary = t`Not found`
  if (launcher.found) {
    summary = games.length > 0 ? games.join(', ') : t`Found, with no supported games yet`
  }
  return (
    <Accordion
      defaultExpanded={open}
      disableGutters={true}
      sx={{
        bgcolor: 'var(--mortar-paper-78)',
        '&::before': { display: 'none' },
      }}
    >
      <AccordionSummary
        expandIcon={<ChevronDown size={18} />}
        sx={{
          '& .MuiAccordionSummary-content': { alignItems: 'center', gap: space.pad, my: 1.25 },
        }}
      >
        <Box sx={{ display: 'flex', flexShrink: 0 }}>
          <LauncherLogo id={launcher.id} size={STATUS_ICON} />
        </Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography sx={{ fontSize: 17, fontWeight: 700 }}>{launcher.name}</Typography>
          <Typography title={summary} sx={{ fontSize: 13, color: 'text.secondary' }} noWrap={true}>
            {summary}
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', mr: 2.5 }}>
          {launcher.found ? (
            <CircleCheck size={STATUS_ICON} color={ok} aria-label={t`Found`} />
          ) : (
            <CircleX size={STATUS_ICON} color={text.secondary} aria-label={t`Not found`} />
          )}
        </Box>
      </AccordionSummary>
      <AccordionDetails sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, pt: 0 }}>
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            bgcolor: 'var(--mortar-raised)',
            borderRadius: '6px',
            overflow: 'hidden',
            '& > * + *': { borderTop: '1px solid var(--mortar-hairline)' },
          }}
        >
          {roots.map((dir) => (
            <FolderRow key={dir} dir={dir} note={t`Found`} found={true} />
          ))}
          {roots.length === 0
            ? looked.map((dir) => (
                <FolderRow key={dir} dir={dir} note={t`Not found`} found={false} />
              ))
            : null}
          {custom.map((dir) => (
            <FolderRow
              key={dir}
              dir={dir}
              note={t`Added by you`}
              found={roots.includes(dir)}
              onRemove={() => run(RemoveLauncherRoot(launcher.id, dir))}
            />
          ))}
        </Box>
        <Box
          sx={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: space.gap }}
        >
          <Button
            variant="text"
            color="inherit"
            size="small"
            startIcon={<RefreshCw size={14} />}
            onClick={refresh}
          >
            {t`Rescan`}
          </Button>
          <Button
            variant={launcher.found ? 'outlined' : 'contained'}
            color={launcher.found ? 'inherit' : 'primary'}
            size="small"
            startIcon={<FolderPlus size={15} />}
            onClick={add}
          >
            {t`Add folder…`}
          </Button>
        </Box>
        {error ? (
          <Typography
            role="alert"
            title={error.details}
            sx={{ fontSize: 13, color: 'error.light' }}
          >
            {error.message}
          </Typography>
        ) : null}
      </AccordionDetails>
    </Accordion>
  )
}

// The launchers Mortar reads, one row each. When the list is wide enough for two columns they stack independently,
// so opening a row only pushes down the rows below it in its own column; otherwise one column in order.
export function LauncherList({
  launchers,
  refresh,
}: {
  launchers: StoreApp[]
  refresh: () => void
}) {
  const columns = [
    { side: 'left', rows: launchers.filter((_, i) => i % 2 === 0) },
    { side: 'right', rows: launchers.filter((_, i) => i % 2 === 1) },
  ]
  // With nothing found at all, the first row starts open so a new user lands on Add folder.
  const first = launchers.every((l) => !l.found) ? launchers[0]?.id : undefined
  const stack = { display: 'flex', flexDirection: 'column', gap: space.gap, minWidth: 0 } as const
  return (
    <Box sx={{ containerType: 'inline-size' }}>
      <Box
        sx={{
          display: 'none',
          gridTemplateColumns: '1fr 1fr',
          alignItems: 'start',
          gap: space.gap,
          [`@container (min-width: ${2 * COLUMN_MIN}px)`]: { display: 'grid' },
        }}
      >
        {columns.map(({ side, rows }) => (
          <Box key={side} sx={stack}>
            {rows.map((l) => (
              <LauncherRow key={l.id} launcher={l} refresh={refresh} open={l.id === first} />
            ))}
          </Box>
        ))}
      </Box>
      <Box sx={{ ...stack, [`@container (min-width: ${2 * COLUMN_MIN}px)`]: { display: 'none' } }}>
        {launchers.map((l) => (
          <LauncherRow key={l.id} launcher={l} refresh={refresh} open={l.id === first} />
        ))}
      </Box>
    </Box>
  )
}
