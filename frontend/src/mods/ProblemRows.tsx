import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import { Info, TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import type { AssetConflict } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/models.ts'
import { useTab } from '../game/tab.ts'
import { useReasonText } from '../i18n/reasonText.ts'
import { space } from '../theme/density.ts'
import { AssetMapDialog } from './AssetMapDialog.tsx'
import { ConflictWhy } from './ConflictWhy.tsx'
import { LinkedText } from './ModLinks.tsx'
import { openPage } from './menu.ts'
import { modLinksOf } from './modLinks.ts'
import { SplitFix } from './problemFix/fixActions.tsx'
import { DriftButtons, FixButton } from './problemFixButtons.tsx'
import { type DismissedRow, isInfoRow, type Row, rowKeys } from './problemGroups.ts'
import { isDismissedRow, useRowText } from './problemText.ts'
import { problemGuideUrl, whyFor } from './problemWhy.ts'

const ICON_PX = 16

function WhyBody({ row }: { row: Row }) {
  const { t } = useLingui()
  const why = whyFor(row)
  const reasonText = useReasonText()
  return (
    <Box sx={{ mt: 0.5, display: 'flex', flexDirection: 'column', gap: 0.75 }}>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', whiteSpace: 'normal' }}>
        {why.text()}
      </Typography>
      <Link
        component="button"
        onClick={() => openPage(problemGuideUrl(why))}
        sx={{ alignSelf: 'flex-start', fontSize: 13 }}
      >
        {t`Read more in the guide`}
      </Link>
      {row.kind === 'loadFailure' ? (
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
          {`${row.loadFailure.line}: ${reasonText(row.loadFailure.code, row.loadFailure.params, row.loadFailure.message)}`}
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

// One quiet row: what is wrong, then Why? and the row's fixes as one split button; the explanation opens beneath.
function ProblemRow({ row, dismissed }: { row: Row; dismissed?: DismissedRow }) {
  const { t } = useLingui()
  const info = isInfoRow(row)
  const { text, note: authorNote } = useRowText()(row)
  const [why, setWhy] = useState(false)
  return (
    <Box
      sx={{
        px: space.pad,
        py: space.gap,
        fontSize: 14,
        '&:not(:first-of-type)': { borderTop: '1px solid var(--mortar-hairline-12)' },
        ...(dismissed ? { opacity: 0.75 } : {}),
      }}
    >
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: `${ICON_PX}px minmax(0, 1fr) auto`,
          columnGap: space.gap,
          alignItems: 'center',
        }}
      >
        <Box
          component="span"
          sx={{ display: 'flex', color: info ? 'text.secondary' : 'warning.main' }}
        >
          {info ? (
            <Info size={ICON_PX} aria-hidden={true} />
          ) : (
            <TriangleAlert size={ICON_PX} aria-hidden={true} />
          )}
        </Box>
        <Box sx={{ minWidth: 0 }}>
          {row.kind === 'missing' ? (
            <Link
              component="button"
              color="inherit"
              onClick={() =>
                useTab.getState().revealLoadOrder(row.missing.id, row.missing.dependentId)
              }
              sx={{
                fontSize: 14,
                whiteSpace: 'normal',
                wordBreak: 'break-word',
                textAlign: 'left',
              }}
            >
              {text}
            </Link>
          ) : (
            <Typography sx={{ fontSize: 14, whiteSpace: 'normal', wordBreak: 'break-word' }}>
              <LinkedText text={text} links={modLinksOf(row)} />
            </Typography>
          )}
          {authorNote === '' ? null : (
            <Typography
              sx={{
                fontSize: 13,
                color: 'text.secondary',
                whiteSpace: 'normal',
                wordBreak: 'break-word',
              }}
            >
              {authorNote}
            </Typography>
          )}
        </Box>
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            flexWrap: 'wrap',
            justifyContent: 'flex-end',
            gap: 0.5,
          }}
        >
          <Button
            size="small"
            color="inherit"
            variant="text"
            aria-expanded={why}
            onClick={() => setWhy(!why)}
            sx={{ flexShrink: 0, color: 'text.secondary' }}
          >
            {t`Why?`}
          </Button>
          <SplitFix>
            {row.kind === 'drift' ? (
              <DriftButtons drift={row.drift} />
            ) : (
              <FixButton problem={row} dismissedToken={dismissed?.token} />
            )}
          </SplitFix>
        </Box>
      </Box>
      {why ? (
        <Box sx={{ pl: `calc(${ICON_PX}px + ${space.gap})`, pb: 0.5 }}>
          <WhyBody row={row} />
        </Box>
      ) : null}
    </Box>
  )
}

// ProblemRows lists one section's rows on one panel.
function ProblemRows({ rows }: { rows: (Row | DismissedRow)[] }) {
  const keys = rowKeys(rows)
  return (
    <Box
      sx={{
        borderRadius: '8px',
        // Light mode keeps rows on opaque paper; a tint over the wallpaper reads as grey there.
        bgcolor: (theme) =>
          theme.palette.mode === 'light'
            ? theme.palette.background.paper
            : 'var(--mortar-overlay-45)',
      }}
    >
      {rows.map((row, i) =>
        isDismissedRow(row) ? (
          <ProblemRow key={keys[i]} row={row.row} dismissed={row} />
        ) : (
          <ProblemRow key={keys[i]} row={row} />
        ),
      )}
    </Box>
  )
}

export { ProblemRows }
