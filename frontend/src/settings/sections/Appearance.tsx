import { Trans, useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Chip, FormControlLabel, Switch } from '@mui/material'
import {
  SetAccent,
  SetTranslucent,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { type AccentName, accents } from '../../theme/accents.ts'
import { useSettings } from '../store.ts'
import { isAccent } from '../theme.ts'

const ignore = () => undefined

export function Appearance() {
  const { t } = useLingui()
  const accent = useSettings((s) => s.accent)
  const translucent = useSettings((s) => s.translucent)
  const cards: { name: AccentName; label: string; note: string }[] = [
    { name: 'sand', label: t`Sand`, note: t`Default` },
    { name: 'moss', label: t`Moss`, note: t`Stardew green` },
    { name: 'copper', label: t`Copper`, note: t`Warm and bold` },
    { name: 'sky', label: t`Sky`, note: t`Cool and calm` },
  ]
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>
        <Trans>Accent colour</Trans>
      </Box>
      <Box
        role="radiogroup"
        aria-label={t`Accent colour`}
        sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, minmax(0, 1fr))', gap: '10px' }}
      >
        {cards.map((card) => {
          const checked = isAccent(accent) && accent === card.name
          return (
            <ButtonBase
              key={card.name}
              role="radio"
              aria-checked={checked}
              onClick={() => {
                SetAccent(card.name).catch(ignore)
              }}
              sx={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'flex-start',
                gap: '10px',
                p: '12px',
                bgcolor: 'rgba(55,55,65,0.9)',
                border: '2px solid',
                borderColor: checked ? '#ffffff' : 'transparent',
                borderRadius: '8px',
                color: '#ffffff',
                textAlign: 'left',
                fontFamily: 'inherit',
              }}
            >
              <Box
                sx={{ width: '100%', height: 44, borderRadius: '6px', bgcolor: accents[card.name] }}
              />
              <Box component="span" sx={{ fontSize: 14, fontWeight: 600 }}>
                {card.label}
              </Box>
              <Box component="span" sx={{ fontSize: 12, color: 'rgba(225,225,230,0.95)' }}>
                {card.note}
              </Box>
            </ButtonBase>
          )
        })}
      </Box>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          p: '14px',
          bgcolor: 'rgba(0,0,0,0.25)',
          borderRadius: '8px',
        }}
      >
        <Button
          variant="contained"
          disableElevation={true}
          tabIndex={-1}
          sx={{ width: 140, height: 44, fontSize: 17, fontWeight: 700 }}
        >
          <Trans>Play</Trans>
        </Button>
        <Chip label={t`3 updates`} color="primary" size="small" sx={{ fontWeight: 700 }} />
        <Box
          component="span"
          sx={{ py: '6px', borderBottom: '2px solid', borderColor: 'primary.main', fontSize: 14 }}
        >
          <Trans>Stardew Valley</Trans>
        </Box>
        <Box
          component="span"
          sx={{ flexGrow: 1, textAlign: 'right', fontSize: 13, color: 'rgba(225,225,230,0.95)' }}
        >
          <Trans>Changes apply right away</Trans>
        </Box>
      </Box>
      <Box>
        <FormControlLabel
          control={
            <Switch
              checked={translucent}
              onChange={(_, on) => {
                SetTranslucent(on).catch(ignore)
              }}
            />
          }
          label={t`Translucent window`}
        />
        <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
          <Trans>Takes effect when Mortar restarts</Trans>
        </Box>
      </Box>
    </Box>
  )
}
