import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { FileArchive } from 'lucide-react'
import { Logo } from '../../brand/Logo.tsx'
import { routeGame, useNav } from '../../nav/store.ts'
import { useProfiles } from '../../profiles/store.ts'
import { saveDiagnostics } from '../../shell/saveDiagnostics.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import credits from '../generated/credits.json' with { type: 'json' }
import { Diagnostics } from './AboutDiagnostics.tsx'

const LICENCE = 'https://github.com/Rethunk-AI/mortar/blob/main/LICENSE'

const body = { color: 'var(--mortar-ink-soft)' }

// Credits flow into as many columns of at least this width as the pane holds.
const CREDIT_COLUMN = '260px'

export function About() {
  const { t } = useLingui()
  const game = useNav((s) => routeGame(s.route) ?? '')
  const profile = useProfiles((s) => s.openId)
  return (
    <Box
      sx={{ display: 'flex', flexDirection: 'column', gap: '12px', fontSize: 14, lineHeight: 1.55 }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <Logo size={40} />
        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
          <Box component="span" sx={{ fontSize: 18, fontWeight: 700 }}>
            {t`Mortar`}
          </Box>
          <Box component="span" sx={{ color: 'var(--mortar-ink-sec)' }}>
            {t`AGPL-3.0 · Rethunk-AI/mortar`}
          </Box>
        </Box>
      </Box>
      <Box sx={{ fontWeight: 600 }}>{t`Built on`}</Box>
      <Box sx={body}>
        {t`SMAPI by Pathoschild (LGPL-3.0) · the Stardew mod dataset by Pathoschild (CC-BY-SA 4.0 / MIT) · Wails · and open-source libraries under their own licences.`}{' '}
        <Link
          component="button"
          onClick={() => Browser.OpenURL(LICENCE).catch(reportUnexpected)}
          sx={{ fontSize: 'inherit', verticalAlign: 'baseline' }}
        >
          {t`Mortar's licence`}
        </Link>
      </Box>
      <Box sx={{ fontWeight: 600 }}>{t`Default background`}</Box>
      <Box sx={body}>
        {t`Fedora 44 default wallpaper (f44-01-night) by the Fedora Design Team, CC-BY-SA 4.0.`}
      </Box>
      <Box sx={{ fontWeight: 600 }}>{t`Credits`}</Box>
      <Box sx={{ ...body, columnWidth: CREDIT_COLUMN, columnGap: 4 }}>
        {credits.map((entry) => (
          <Box
            key={entry.name}
            sx={{ display: 'flex', gap: '6px', mb: '6px', breakInside: 'avoid', minWidth: 0 }}
          >
            <Link
              component="button"
              onClick={() => Browser.OpenURL(entry.url).catch(reportUnexpected)}
              title={entry.name}
              sx={{
                minWidth: 0,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                fontSize: 'inherit',
                verticalAlign: 'baseline',
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
      <Diagnostics />
      <Box sx={{ fontSize: 13, color: 'text.secondary', lineHeight: 1.5 }}>
        {t`Save diagnostics… writes a redacted zip for a bug report. Run checks for a live summary.`}
      </Box>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<FileArchive size={16} />}
        onClick={() => saveDiagnostics(game, profile)}
        sx={{ alignSelf: 'flex-start', whiteSpace: 'nowrap' }}
      >
        {t`Save diagnostics…`}
      </Button>
    </Box>
  )
}
