import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ButtonBase,
  Chip,
  FormControl,
  FormHelperText,
  InputLabel,
  MenuItem,
  Select,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import { ImagePlus, RotateCcw } from 'lucide-react'
import { useId } from 'react'
import {
  ChooseBackgroundImage,
  SetAccent,
  SetBackground,
  SetBackgroundImage,
  SetLanguage,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { availableLocales } from '../../i18n/locales.ts'
import { type AccentName, accents } from '../../theme/accents.ts'
import { errorText } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { useSettings } from '../store.ts'
import { isAccent } from '../theme.ts'

export function Appearance() {
  const { t } = useLingui()
  const languageLabelId = useId()
  const accent = useSettings((s) => s.accent)
  const background = useSettings((s) => s.background)
  const backgroundImage = useSettings((s) => s.backgroundImage)
  const language = useSettings((s) => s.language)
  const push = useToasts((s) => s.push)
  const reportFailure = (err: unknown) => {
    const body = errorText(err)
    push({ kind: 'error', title: t`Couldn't save that setting`, ...(body ? { body } : {}) })
  }
  const cards: { name: AccentName; label: string; note: string }[] = [
    { name: 'sand', label: t`Sand`, note: t`Default` },
    { name: 'moss', label: t`Moss`, note: t`Earthy green` },
    { name: 'copper', label: t`Copper`, note: t`Warm and bold` },
    { name: 'sky', label: t`Sky`, note: t`Cool and calm` },
  ]
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <FormControl size="small" disabled={availableLocales.length === 1}>
        <InputLabel id={languageLabelId}>{t`Language`}</InputLabel>
        <Select
          labelId={languageLabelId}
          value={language}
          label={t`Language`}
          onChange={(event) => {
            SetLanguage(event.target.value).catch(reportFailure)
          }}
        >
          <MenuItem value="">{t`System default`}</MenuItem>
          {availableLocales.map((locale) => (
            <MenuItem key={locale} value={locale}>
              {new Intl.DisplayNames([locale], { type: 'language' }).of(locale) ?? locale}
            </MenuItem>
          ))}
        </Select>
        {availableLocales.length === 1 ? (
          <FormHelperText>{t`More languages are coming.`}</FormHelperText>
        ) : null}
      </FormControl>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Accent colour`}</Box>
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
                SetAccent(card.name).catch(reportFailure)
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
          tabIndex={-1}
          sx={{ width: 140, height: 44, fontSize: 17, fontWeight: 700 }}
        >
          {t`Play`}
        </Button>
        <Chip label={t`3 updates`} color="primary" size="small" sx={{ fontWeight: 700 }} />
        <Box
          component="span"
          sx={{ py: '6px', borderBottom: '2px solid', borderColor: 'primary.main', fontSize: 14 }}
        >
          {t`Stardew Valley`}
        </Box>
        <Box
          component="span"
          sx={{ flexGrow: 1, textAlign: 'right', fontSize: 13, color: 'rgba(225,225,230,0.95)' }}
        >
          {t`Changes apply right away`}
        </Box>
      </Box>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Background`}</Box>
      <ToggleButtonGroup
        exclusive={true}
        size="small"
        value={background}
        aria-label={t`Background`}
        onChange={(_, next: string | null) => {
          if (next) {
            SetBackground(next).catch(reportFailure)
          }
        }}
        sx={{ alignSelf: 'flex-start', '& .MuiToggleButton-root': { whiteSpace: 'nowrap' } }}
      >
        <ToggleButton value="image">{t`Image`}</ToggleButton>
        <ToggleButton value="desktop">{t`Desktop`}</ToggleButton>
        <ToggleButton value="solid">{t`Solid`}</ToggleButton>
      </ToggleButtonGroup>
      {background === 'image' ? (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <Box
            component="img"
            alt={t`Background preview`}
            src={`/backdrop?v=${encodeURIComponent(backgroundImage)}`}
            sx={{ width: 160, height: 90, objectFit: 'cover', borderRadius: '6px' }}
          />
          <Button
            variant="outlined"
            startIcon={<ImagePlus size={16} />}
            onClick={() => {
              ChooseBackgroundImage().catch(reportFailure)
            }}
          >
            {t`Choose image…`}
          </Button>
          <Button
            variant="outlined"
            startIcon={<RotateCcw size={16} />}
            disabled={backgroundImage === ''}
            onClick={() => {
              SetBackgroundImage('').catch(reportFailure)
            }}
          >
            {t`Reset to default`}
          </Button>
        </Box>
      ) : null}
    </Box>
  )
}
