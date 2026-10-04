import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSwitch } from '../PrefControls.tsx'
import { PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { specByKey, usePrefSpecs } from '../prefSpecs.ts'
import { prefAsBool, prefRaw } from '../prefValue.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

const CHANNEL_WIDTH = 96

interface EventRow {
  label: string
  toast: string
  desktop: string
}

function ChannelSwitch({ prefKey, label }: { prefKey: string; label: string }) {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const settings = useSettings()
  const spec = specByKey(usePrefSpecs(), prefKey)
  if (!spec) {
    return <Box sx={{ width: CHANNEL_WIDTH }} />
  }
  return (
    <Box sx={{ width: CHANNEL_WIDTH, display: 'flex', justifyContent: 'center' }}>
      <PrefSwitch
        checked={prefAsBool(prefRaw(settings, spec), spec)}
        label={label}
        onChange={(on) =>
          persist(() => SetByKey(prefKey, String(on), ''), push, t`Could not save that setting`)
        }
      />
    </Box>
  )
}

function ChannelHeader() {
  const { t } = useLingui()
  const cell = {
    width: CHANNEL_WIDTH,
    textAlign: 'center',
    fontSize: 12,
    fontWeight: 700,
    letterSpacing: '0.06em',
    textTransform: 'uppercase',
    color: 'text.secondary',
  } as const
  return (
    <Box component="span" sx={{ display: 'flex', alignItems: 'baseline' }}>
      <Box component="span" sx={{ flex: 1 }}>{t`When`}</Box>
      <Box component="span" sx={{ display: 'flex', pr: 2.5 }}>
        <Box component="span" sx={cell}>{t`In Mortar`}</Box>
        <Box component="span" sx={cell}>{t`Desktop`}</Box>
      </Box>
    </Box>
  )
}

export function Notifications() {
  const { t } = useLingui()
  const events: EventRow[] = [
    {
      label: t`A download finishes`,
      toast: 'notifyDownloadFinished',
      desktop: 'desktopDownloadFinished',
    },
    { label: t`A download fails`, toast: 'notifyDownloadFailed', desktop: 'desktopDownloadFailed' },
    { label: t`The game crashes`, toast: 'notifyRunCrashed', desktop: 'desktopRunCrashed' },
    { label: t`Mod updates are found`, toast: 'notifyModUpdates', desktop: 'desktopModUpdates' },
  ]
  return (
    <>
      <SettingsSection title={<ChannelHeader />}>
        {events.map((e) => (
          <SettingRow key={e.toast} label={e.label}>
            <Box sx={{ display: 'flex' }}>
              <ChannelSwitch prefKey={e.toast} label={t`${e.label}: in Mortar`} />
              <ChannelSwitch prefKey={e.desktop} label={t`${e.label}: desktop notification`} />
            </Box>
          </SettingRow>
        ))}
      </SettingsSection>
      <SettingsSection title={t`Mod updates`}>
        <PrefKeys keys={['updateDigest']} />
      </SettingsSection>
    </>
  )
}
