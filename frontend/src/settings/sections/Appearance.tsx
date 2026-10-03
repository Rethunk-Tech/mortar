import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Chip, ToggleButton, ToggleButtonGroup } from '@mui/material'
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
import { errorText } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSelect } from '../PrefControls.tsx'
import { PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
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
          persist(() => SetListSort(column, dir), push, t`Couldn't save that setting`)
        }}
        options={options}
      />
    </SettingRow>
  )
}

export function Appearance() {
  const { t } = useLingui()
  const accent = useSettings((s) => s.accent)
  const background = useSettings((s) => s.background)
  const backgroundImage = useSettings((s) => s.backgroundImage)
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
      <SettingsSection title={t`Theme`}>
        <PrefKeys keys={['theme']} />
      </SettingsSection>
      <SettingsSection title={t`Accent colour`}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', p: 2 }}>
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
                    bgcolor: 'var(--mortar-raised)',
                    border: '2px solid',
                    borderColor: checked ? 'var(--mortar-ink)' : 'transparent',
                    borderRadius: '8px',
                    color: 'var(--mortar-ink)',
                    textAlign: 'left',
                    fontFamily: 'inherit',
                  }}
                >
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
              bgcolor: 'var(--mortar-overlay-25)',
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
              sx={{
                py: '6px',
                borderBottom: '2px solid',
                borderColor: 'primary.main',
                fontSize: 14,
              }}
            >
              {t`Stardew Valley`}
            </Box>
            <Box
              component="span"
              sx={{ flexGrow: 1, textAlign: 'right', fontSize: 13, color: 'var(--mortar-ink-sec)' }}
            >
              {t`Changes apply right away`}
            </Box>
          </Box>
        </Box>
      </SettingsSection>
      <SettingsSection title={t`Background`}>
        <SettingRow label={t`Background`}>
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
            sx={{ '& .MuiToggleButton-root': { whiteSpace: 'nowrap' } }}
          >
            <ToggleButton value="image">{t`Image`}</ToggleButton>
            <ToggleButton value="desktop">{t`Desktop`}</ToggleButton>
            <ToggleButton value="solid">{t`Solid`}</ToggleButton>
          </ToggleButtonGroup>
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
                {t`Reset to default`}
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
