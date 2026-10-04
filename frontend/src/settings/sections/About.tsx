import { useLingui } from '@lingui/react/macro'
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, Link } from '@mui/material'
import { useState } from 'react'
import { Logo } from '../../brand/Logo.tsx'
import { openPage } from '../../mods/menu.ts'
import { paper } from '../../mods/paper.ts'
import credits from '../generated/credits.json' with { type: 'json' }
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { Diagnostics } from './AboutDiagnostics.tsx'

const LICENCE = 'https://github.com/Rethunk-AI/mortar/blob/main/LICENSE'

const body = { color: 'var(--mortar-ink-soft)' }

// Credits flow into as many columns of at least this width as the pane holds.
const CREDIT_COLUMN = '260px'

function CreditList() {
  return (
    <Box
      sx={{
        ...body,
        display: 'grid',
        gridTemplateColumns: `repeat(auto-fill, minmax(${CREDIT_COLUMN}, 1fr))`,
        columnGap: 4,
        rowGap: '6px',
        fontSize: 14,
      }}
    >
      {credits.map((entry) => (
        <Box
          key={entry.name}
          sx={{ display: 'flex', alignItems: 'baseline', gap: '6px', minWidth: 0 }}
        >
          <Link
            component="button"
            onClick={() => openPage(entry.url)}
            title={entry.name}
            sx={{
              minWidth: 0,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              fontSize: 'inherit',
            }}
          >
            {entry.name}
          </Link>
          <Box component="span" sx={{ flexShrink: 0, whiteSpace: 'nowrap' }}>
            {`· ${entry.licence}`}
          </Box>
        </Box>
      ))}
    </Box>
  )
}

export function About() {
  const { t } = useLingui()
  const [creditsOpen, setCreditsOpen] = useState(false)
  return (
    <>
      <SettingsSection title={t`Mortar`}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, px: 2.5, py: 2 }}>
          <Logo size={40} />
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
            <Box component="span" sx={{ fontSize: 18, fontWeight: 700 }}>
              {t`Mortar`}
            </Box>
            <Box component="span" sx={{ color: 'var(--mortar-ink-sec)', fontSize: 14 }}>
              {t`AGPL-3.0 · Rethunk-AI/mortar`}
            </Box>
          </Box>
          <Button variant="outlined" onClick={() => openPage(LICENCE)}>
            {t`Licence`}
          </Button>
        </Box>
        <SettingRow
          label={t`Open-source licences`}
          description={t`Libraries, icons and artwork built into Mortar, with their licences.`}
        >
          <Button variant="outlined" onClick={() => setCreditsOpen(true)}>
            {t`View (${credits.length})`}
          </Button>
        </SettingRow>
      </SettingsSection>
      <Diagnostics />
      <Dialog
        open={creditsOpen}
        onClose={() => setCreditsOpen(false)}
        maxWidth="md"
        fullWidth={true}
        scroll="paper"
        transitionDuration={0}
        slotProps={{ paper }}
      >
        <DialogTitle>{t`Open-source licences`}</DialogTitle>
        <DialogContent dividers={true}>
          <CreditList />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreditsOpen(false)}>{t`Close`}</Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
