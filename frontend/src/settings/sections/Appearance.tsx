import { useLingui } from '@lingui/react/macro'
import { Box, Button, FormControlLabel, Radio, RadioGroup } from '@mui/material'
import { ImagePlus, RotateCcw } from 'lucide-react'
import {
  ChooseBackgroundImage,
  SetAccent,
  SetBackground,
  SetBackgroundImage,
  SetListSort,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { columnLabel } from '../../mods/columnLabel.ts'
import { LIST_COLUMN_IDS, sanitizeListSort } from '../../mods/listColumns.ts'
import { type AccentName, accents } from '../../theme/accents.ts'
import { reportError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSegmented, PrefSelect } from '../PrefControls.tsx'
import { PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { Searchable, SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { isAccent } from '../theme.ts'

function DefaultSort() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
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
          persist(() => SetListSort(column, dir), push, t`Could not save that setting`)
        }}
        options={options}
        label={t`Default sort`}
      />
    </SettingRow>
  )
}

const visuallyHidden = {
  position: 'absolute',
  opacity: 0,
  width: 1,
  height: 1,
  p: 0,
  overflow: 'hidden',
} as const

export function Appearance() {
  const { t } = useLingui()
  const accent = useSettings((s) => s.accent)
  const background = useSettings((s) => s.background)
  const backgroundImage = useSettings((s) => s.backgroundImage)
  const reportFailure = reportError(t`Could not save that setting`)
  const cards: { name: AccentName; label: string; note: string }[] = [
    { name: 'sand', label: t`Sand`, note: t`Default` },
    { name: 'moss', label: t`Moss`, note: t`Earthy green` },
    { name: 'copper', label: t`Copper`, note: t`Warm and bold` },
    { name: 'sky', label: t`Sky`, note: t`Cool and calm` },
    { name: 'rose', label: t`Rose`, note: t`Soft and warm` },
    { name: 'lavender', label: t`Lavender`, note: t`Gentle violet` },
    { name: 'teal', label: t`Teal`, note: t`Fresh and clear` },
    { name: 'slate', label: t`Slate`, note: t`Quiet grey-blue` },
  ]
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <SettingsSection title={t`Theme`}>
        <PrefKeys keys={['theme']} />
      </SettingsSection>
      <SettingsSection title={t`Accent colour`}>
        <Searchable terms={`${t`Accent colour`} ${t`colour`} ${t`color`} ${t`theme`}`}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', p: 2 }}>
            <RadioGroup
              aria-label={t`Accent colour`}
              value={isAccent(accent) ? accent : ''}
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
                          height: 44,
                          borderRadius: '6px',
                          bgcolor: accents[card.name],
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
                    '&:has(input:focus-visible)': {
                      outline: '2px solid',
                      outlineColor: 'primary.main',
                    },
                  }}
                />
              ))}
            </RadioGroup>
          </Box>
        </Searchable>
      </SettingsSection>
      <SettingsSection title={t`Background`}>
        <SettingRow label={t`Background`}>
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
                {t`Use default`}
              </Button>
            </Box>
          </SettingRow>
        ) : null}
      </SettingsSection>
      <SettingsSection title={t`Display`}>
        <PrefKeys keys={['dates', 'density', 'reduceMotion', 'profileHero']} />
      </SettingsSection>
      <SettingsSection title={t`Mods view`}>
        <PrefKeys keys={['defaultModsView', 'gridCardSize', 'showAuthorOnCards', 'listGroupBy']} />
        <DefaultSort />
      </SettingsSection>
      <SettingsSection title={t`Sidebar`}>
        <PrefKeys keys={['profileOrder', 'sidebarBadges']} />
      </SettingsSection>
    </Box>
  )
}
