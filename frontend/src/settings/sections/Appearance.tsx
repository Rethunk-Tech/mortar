import { useLingui } from '@lingui/react/macro'
import {
  Autocomplete,
  Box,
  Button,
  FormControlLabel,
  Radio,
  RadioGroup,
  TextField,
  Typography,
} from '@mui/material'
import { ImagePlus, RotateCcw } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  ChooseBackgroundImage,
  InstalledFonts,
  SetAccent,
  SetBackground,
  SetBackgroundImage,
  SetByKey,
  SetListSort,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { columnLabel } from '../../mods/columnLabel.ts'
import { LIST_COLUMN_IDS, sanitizeListSort } from '../../mods/listColumns.ts'
import { BackdropImage } from '../../shell/BackdropImage.tsx'
import { visuallyHidden } from '../../shell/visuallyHidden.ts'
import { space } from '../../theme/density.ts'
import { PAD_FOCUS } from '../../theme/theme.ts'
import { reportError, reportUnexpected } from '../../toasts/report.ts'
import { PrefSegmented, PrefSelect } from '../PrefControls.tsx'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { Searchable, SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { accentColour, isAccent, SYSTEM_ACCENT, useSystemAccent } from '../theme.ts'

// An image above this is decoded whole on every start, and an animated GIF is never shrunk, so it costs memory and GPU.
const LARGE_BACKGROUND_BYTES = 16_777_216 // 16 MiB

// The chosen background's size, read from the page's own request for it; 0 until known.
function useBackgroundBytes(src: string): number {
  const [bytes, setBytes] = useState({ src: '', n: 0 })
  useEffect(() => {
    const abort = new AbortController()
    fetch(src, { signal: abort.signal })
      .then((res) => {
        setBytes({ src, n: Number(res.headers.get('content-length')) || 0 })
        abort.abort()
      })
      .catch(() => undefined)
    return () => abort.abort()
  }, [src])
  return bytes.src === src ? bytes.n : 0
}

// Its own component, so the size arriving re-renders this line and not the whole Appearance page.
function LargeBackgroundNote({ src }: { src: string }) {
  const { t } = useLingui()
  const bytes = useBackgroundBytes(src)
  return bytes > LARGE_BACKGROUND_BYTES ? (
    <Typography sx={{ maxWidth: 420, fontSize: 12, textAlign: 'right', color: 'warning.main' }}>
      {t`This image is ${formatBytes(bytes)}, which costs memory and GPU while Mortar is open. The same picture or animation saved as WebP is often a tenth of the size.`}
    </Typography>
  ) : null
}

function DefaultSort() {
  const { t } = useLingui()
  const stored = sanitizeListSort(
    useSettings((s) => s.listSortColumn),
    useSettings((s) => s.listSortDir),
  )
  const options = LIST_COLUMN_IDS.filter((id) => id !== 'on').flatMap((id) => {
    const name = columnLabel(id)
    return [
      { value: `${id}:asc`, label: t`${name}, ascending` },
      { value: `${id}:desc`, label: t`${name}, descending` },
    ]
  })
  return (
    <SettingRow label={t`Default sort`}>
      <PrefSelect
        value={`${stored.column}:${stored.dir}`}
        onChange={(v) => {
          const [column = 'name', dir = 'asc'] = v.split(':')
          persist(() => SetListSort(column, dir), t`Could not save that setting`)
        }}
        options={options}
        label={t`Default sort`}
      />
    </SettingRow>
  )
}

// The accent swatch's height: a picture of the colour, not a control, so it does not follow the density scale.
const SWATCH_PX = 44

function FontPicker() {
  const { t } = useLingui()
  const font = useSettings((s) => s.font)
  const [families, setFamilies] = useState<string[] | null>(null)
  // The installed fonts are listed when the picker opens, not whenever settings mount.
  const loadFamilies = () => {
    if (families === null) {
      InstalledFonts()
        .then((list) => setFamilies(list ?? []))
        .catch(reportUnexpected)
    }
  }
  return (
    <Searchable terms={`${t`App font`} ${t`Font`} ${t`Typeface`}`}>
      <SettingRow label={t`App font`}>
        <Autocomplete
          size="small"
          sx={{ width: 280 }}
          options={families ?? (font === '' ? [] : [font])}
          onOpen={loadFamilies}
          value={font === '' ? null : font}
          onChange={(_, family) => {
            persist(() => SetByKey('font', family ?? '', ''), t`Could not save that setting`)
          }}
          renderInput={(params) => <TextField {...params} placeholder={t`Bundled font`} />}
        />
      </SettingRow>
    </Searchable>
  )
}

// Its own component: the desktop's accent colour arrives after mount and re-renders only the swatches, not the page.
function AccentPicker() {
  const { t } = useLingui()
  const accent = useSettings((s) => s.accent)
  const reportFailure = reportError(t`Could not save that setting`)
  const systemAccent = useSystemAccent()
  const cards: { name: string; label: string; note: string }[] = [
    { name: 'sand', label: t`Sand`, note: t`Default` },
    { name: 'moss', label: t`Moss`, note: t`Earthy green` },
    { name: 'copper', label: t`Copper`, note: t`Warm and bold` },
    { name: 'sky', label: t`Sky`, note: t`Cool and calm` },
    { name: 'rose', label: t`Rose`, note: t`Soft and warm` },
    { name: 'lavender', label: t`Lavender`, note: t`Gentle violet` },
    { name: 'teal', label: t`Teal`, note: t`Fresh and clear` },
    { name: 'slate', label: t`Slate`, note: t`Quiet grey-blue` },
    { name: SYSTEM_ACCENT, label: t`System`, note: t`Follows your desktop` },
  ]
  return (
    <Searchable terms={`${t`Accent colour`} ${t`Colour`} ${t`color`} ${t`Theme`}`}>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', p: space.pad }}>
        <RadioGroup
          aria-label={t`Accent colour`}
          value={isAccent(accent) || accent === SYSTEM_ACCENT ? accent : ''}
          onChange={(_, name) => {
            SetAccent(name).catch(reportFailure)
          }}
          sx={{
            display: 'grid',
            gridTemplateColumns: 'repeat(4, minmax(0, 1fr))',
            gap: '10px',
          }}
        >
          {cards.map((card) => (
            <FormControlLabel
              key={card.name}
              value={card.name}
              // The swatch card is the visible control; the radio stays in the tab order for arrow keys.
              control={<Radio sx={visuallyHidden} />}
              label={
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                  <Box
                    sx={{
                      width: '100%',
                      height: SWATCH_PX,
                      borderRadius: '6px',
                      bgcolor: accentColour(card.name, systemAccent),
                    }}
                  />
                  <Box component="span" sx={{ fontSize: 14, fontWeight: 600 }}>
                    {card.label}
                  </Box>
                  <Box component="span" sx={{ fontSize: 12, color: 'var(--mortar-ink-sec)' }}>
                    {card.note}
                  </Box>
                </Box>
              }
              sx={{
                m: 0,
                p: '12px',
                position: 'relative',
                bgcolor: 'var(--mortar-raised)',
                border: '2px solid transparent',
                borderRadius: '8px',
                color: 'var(--mortar-ink)',
                '& .MuiFormControlLabel-label': { width: '100%' },
                '&:has(input:checked)': { borderColor: 'var(--mortar-ink)' },
                [`&:has(input:focus-visible), ${PAD_FOCUS} &:has(input:focus)`]: {
                  outline: '2px solid',
                  outlineColor: 'primary.main',
                },
              }}
            />
          ))}
        </RadioGroup>
      </Box>
    </Searchable>
  )
}

export function Appearance() {
  const { t } = useLingui()
  const background = useSettings((s) => s.background)
  const backgroundImage = useSettings((s) => s.backgroundImage)
  const reportFailure = reportError(t`Could not save that setting`)
  const backdropSrc = `/backdrop?mode=image&v=${encodeURIComponent(backgroundImage)}`
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
      <SettingsSection title={t`Theme`}>
        <PrefByKey prefKey="theme" hideTitle={true} />
      </SettingsSection>
      <SettingsSection title={t`Accent colour`}>
        <AccentPicker />
      </SettingsSection>
      <SettingsSection title={t`Font`}>
        <FontPicker />
      </SettingsSection>
      <SettingsSection title={t`Background`}>
        <SettingRow label={t`Window background`}>
          <PrefSegmented
            value={background}
            label={t`Background`}
            onChange={(next) => {
              SetBackground(next).catch(reportFailure)
            }}
            options={[
              { value: 'image', label: t`Image` },
              { value: 'desktop', label: t`Desktop` },
              { value: 'solid', label: t`Solid` },
            ]}
          />
        </SettingRow>
        {background === 'image' ? (
          <SettingRow label={t`Image`}>
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <BackdropImage
                  alt={t`Background preview`}
                  src={backdropSrc}
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
                  {t`Use default`}
                </Button>
              </Box>
              {backgroundImage === '' ? null : <LargeBackgroundNote src={backdropSrc} />}
            </Box>
          </SettingRow>
        ) : null}
      </SettingsSection>
      <SettingsSection title={t`Display`}>
        <PrefKeys
          keys={['dates', 'density', 'reduceMotion', 'profileHero', 'centerConfigControls']}
        />
      </SettingsSection>
      <SettingsSection title={t`Mods view`}>
        <PrefKeys
          keys={[
            'defaultModsView',
            'gridCardSize',
            'showAuthorOnCards',
            'listGroupBy',
            'implicitModsView',
          ]}
        />
        <DefaultSort />
      </SettingsSection>
      <SettingsSection title={t`Sidebar`}>
        <PrefKeys keys={['profileOrder', 'sidebarBadges']} />
      </SettingsSection>
    </Box>
  )
}
