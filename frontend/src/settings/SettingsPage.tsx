import { useLingui } from '@lingui/react/macro'
import {
  Bell,
  Download,
  HardDrive,
  Info,
  Keyboard,
  Package,
  Palette,
  RefreshCw,
  Rocket,
  SlidersHorizontal,
  UserRound,
} from 'lucide-react'
import type { ReactNode } from 'react'
import { type SettingsSection, useNav } from '../nav/store.ts'
import { SettingsShell, type ShellPage } from './SettingsShell.tsx'
import { About } from './sections/About.tsx'
import { Appearance } from './sections/Appearance.tsx'
import { Downloads } from './sections/Downloads.tsx'
import { General } from './sections/General.tsx'
import { Launchers } from './sections/Launchers.tsx'
import { ModsProfiles } from './sections/ModsProfiles.tsx'
import { NexusMods } from './sections/NexusMods.tsx'
import { Notifications } from './sections/Notifications.tsx'
import { ResetAllShortcuts, Shortcuts } from './sections/Shortcuts.tsx'
import { Storage } from './sections/Storage.tsx'
import { Updates } from './sections/Updates.tsx'

export function SettingsPage({ section }: { section: SettingsSection }) {
  const { t } = useLingui()
  const closeSettings = useNav((s) => s.closeSettings)
  const pages: ShellPage<SettingsSection>[] = [
    { id: 'general', label: t`General`, icon: SlidersHorizontal },
    { id: 'appearance', label: t`Appearance`, icon: Palette, groupEnd: true },
    { id: 'mods', label: t`Mods and profiles`, icon: Package },
    { id: 'downloads', label: t`Downloads`, icon: Download },
    { id: 'nexus', label: t`Nexus account`, icon: UserRound },
    { id: 'updates', label: t`Updates`, icon: RefreshCw },
    { id: 'notifications', label: t`Notifications`, icon: Bell, groupEnd: true },
    { id: 'storage', label: t`Storage`, icon: HardDrive },
    { id: 'launchers', label: t`Launchers`, icon: Rocket, groupEnd: true },
    { id: 'shortcuts', label: t`Shortcuts`, icon: Keyboard },
    { id: 'about', label: t`About`, icon: Info },
  ]
  const body: Record<SettingsSection, ReactNode> = {
    general: <General />,
    appearance: <Appearance />,
    mods: <ModsProfiles />,
    downloads: <Downloads />,
    nexus: <NexusMods />,
    updates: <Updates />,
    notifications: <Notifications />,
    storage: <Storage />,
    launchers: <Launchers />,
    shortcuts: <Shortcuts />,
    about: <About />,
  }
  return (
    <SettingsShell
      title={t`Settings`}
      backLabel={t`Back`}
      onBack={closeSettings}
      pages={pages}
      current={section}
      onPage={(id) => useNav.getState().openSettings(id)}
      render={(id) => body[id]}
      actions={{ shortcuts: <ResetAllShortcuts /> }}
    />
  )
}
