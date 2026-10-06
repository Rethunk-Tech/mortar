import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Link, Typography } from '@mui/material'
import { ChevronDown, ChevronRight, Info, TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import type { AssetConflict } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/models.ts'
import { useTab } from '../game/tab.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { AssetMapDialog } from './AssetMapDialog.tsx'
import { ConflictWhy } from './ConflictWhy.tsx'
import { LinkedText } from './ModLinks.tsx'
import { openPage } from './menu.ts'
import { modLinksOf } from './modLinks.ts'
import { DriftButtons, FixButton } from './problemFixButtons.tsx'
import { type DismissedRow, isInfoRow, type Row } from './problemGroups.ts'
import { isDismissedRow, useRowText } from './problemText.ts'
import { problemGuideUrl, problemWhy } from './problemWhy.ts'

function WhyBody({ row }: { row: Row }) {
  const { t } = useLingui()
  return (
    <Box sx={{ mt: 0.5, display: 'flex', flexDirection: 'column', gap: 0.75 }}>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', whiteSpace: 'normal' }}>
        {problemWhy[row.kind].text()}
      </Typography>
      <Link
        component="button"
        onClick={() => openPage(problemGuideUrl(row.kind))}
        sx={{ alignSelf: 'flex-start', fontSize: 13 }}
      >
        {t`Read more in the guide`}
      </Link>
      {row.kind === 'loadFailure' ? (
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
          {`${row.loadFailure.line}: ${row.loadFailure.message}`}
        </Typography>
      ) : null}
      {row.kind === 'asset' ? <AssetConflictWhy asset={row.asset} /> : null}
    </Box>
  )
}

function AssetConflictWhy({ asset }: { asset: AssetConflict }) {
  const { t } = useLingui()
  const [mapOpen, setMapOpen] = useState(false)
  return (
    <>
      <ConflictWhy asset={asset} />
      <Link
        component="button"
        onClick={() => setMapOpen(true)}
        sx={{ alignSelf: 'flex-start', fontSize: 13 }}
      >
        {t`Show conflicts`}
      </Link>
      {mapOpen ? (
        <AssetMapDialog open={true} focus={asset.target} onClose={() => setMapOpen(false)} />
      ) : null}
    </>
  )
}

function ProblemRow({ row, dismissed }: { row: Row; dismissed?: DismissedRow }) {
  const { t } = useLingui()
  const info = isInfoRow(row)
  const { text, note: authorNote } = useRowText()(row)
  const [why, setWhy] = useState(false)
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 1.25,
        flexShrink: 0,
        pl: 1.5,
        pr: 0.75,
        py: 1,
        fontSize: 14,
        // Light mode keeps cards on opaque paper (a tint over the wallpaper reads as grey); the warning border and icon
        // still set conflicts apart.
        bgcolor: (theme) => {
          if (theme.palette.mode === 'light') {
            return theme.palette.background.paper
          }
          return info ? 'var(--mortar-overlay-45)' : calloutFill('warning')(theme)
        },
        border: '1px solid',
        borderColor: info ? 'transparent' : calloutLine('warning'),
        borderRadius: '6px',
        ...(dismissed ? { opacity: 0.75 } : {}),
      }}
    >
      <Box
        component="span"
        sx={{
          display: 'flex',
          flexShrink: 0,
          mt: 0.25,
          color: info ? 'text.secondary' : 'warning.main',
        }}
      >
        {info ? (
          <Info size={16} aria-hidden={true} />
        ) : (
          <TriangleAlert size={16} aria-hidden={true} />
        )}
      </Box>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        {row.kind === 'missing' ? (
          <Link
            component="button"
            color="inherit"
            onClick={() =>
              useTab.getState().revealLoadOrder(row.missing.id, row.missing.dependentId)
            }
            sx={{ fontSize: 14, whiteSpace: 'normal', wordBreak: 'break-word', textAlign: 'left' }}
          >
            {text}
          </Link>
        ) : (
          <Typography sx={{ fontSize: 14, whiteSpace: 'normal', wordBreak: 'break-word' }}>
            <LinkedText text={text} links={modLinksOf(row)} />
          </Typography>
        )}
        {authorNote === '' ? null : (
          <Typography sx={{ mt: 0.5, fontSize: 13, color: 'text.secondary', whiteSpace: 'normal' }}>
            {authorNote}
          </Typography>
        )}
        <Box sx={{ mt: 0.75 }}>
          <ButtonBase
            onClick={() => setWhy(!why)}
            aria-expanded={why}
            sx={{ display: 'flex', alignItems: 'center', gap: 0.5, borderRadius: '4px' }}
          >
            {why ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            <Typography
              component="span"
              sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary' }}
            >
              {t`Why?`}
            </Typography>
          </ButtonBase>
          {why ? <WhyBody row={row} /> : null}
        </Box>
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, justifyContent: 'flex-end' }}>
        {row.kind === 'drift' ? (
          <DriftButtons drift={row.drift} />
        ) : (
          <FixButton problem={row} dismissedToken={dismissed?.token} />
        )}
      </Box>
    </Box>
  )
}

// ProblemRows lists the cards of the chosen section.
function ProblemRows({ rows }: { rows: (Row | DismissedRow)[] }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      {rows.map((row) =>
        isDismissedRow(row) ? (
          <ProblemRow key={row.token} row={row.row} dismissed={row} />
        ) : (
          <ProblemRow key={JSON.stringify(row)} row={row} />
        ),
      )}
    </Box>
  )
}

export { ProblemRows }
