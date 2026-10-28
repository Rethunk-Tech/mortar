import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { Application } from '@wailsio/runtime'
import {
  Bug,
  ChevronDown,
  Code2,
  Download,
  FolderOpen,
  Info,
  LifeBuoy,
  Lightbulb,
  LogIn,
  LogOut,
  RefreshCw,
  Settings,
  Sparkles,
  Timer,
} from 'lucide-react'
import { memo, type ReactNode, useEffect } from 'react'
import { SignOut } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { OpenDataFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { RestartProfilingStartup } from '../../bindings/github.com/Rethunk-Tech/mortar/quitservice.ts'
import { DiscordLogo } from '../brand/Discord.tsx'
import { Logo } from '../brand/Logo.tsx'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useConsole } from '../console/store.ts'
import { compact } from '../game/compact.ts'
import { useTab } from '../game/tab.ts'
import { openPage } from '../mods/menu.ts'
import { openSettings, routeGame, useNav } from '../nav/store.ts'
import { useQueue } from '../queue/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { useMortarUpdate } from '../settings/updates.ts'
import { useShortcutHint } from '../settings/useShortcutHint.ts'
import { space } from '../theme/density.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { showWhatsNew } from '../updates/whatsNew.ts'
import { checkForUpdates } from './checkForUpdates.ts'
import { GO } from './destination.ts'
import { useUpdatesOfflineReason } from './offlineText.ts'
import { reportBug } from './reportBug.ts'
import { suggestFeature } from './suggest.ts'
import { MenuHeading, MenuRule, TitleMenu, TitleMenuItem } from './TitleMenu.tsx'
import { Tooltip } from './Tooltip.tsx'
import { closeTitleMenu, useTitleMenu } from './titleMenus.ts'

const ICON_PX = 16
const SOURCE = 'https://github.com/Rethunk-Tech/mortar'

// A disabled row cannot take hover, so its reason rides on a wrapper.
function Reason({ reason, children }: { reason: string; children: ReactNode }) {
  return reason === '' ? (
    children
  ) : (
    <Tooltip title={reason} placement="left" describeChild={true}>
      <span>{children}</span>
    </Tooltip>
  )
}

function NexusLines({ close }: { close: () => void }) {
  const { t } = useLingui()
  const signedIn = useNexus((s) => s.signedIn)
  const name = useNexus((s) => s.name)
  const premium = useNexus((s) => s.premium)
  return (
    <>
      <Box sx={{ px: '14px', py: '6px', fontSize: 'base.fontSize' }}>
        {[sourceLabel('nexus'), signedIn ? t`Signed in as ${name}` : t`Not signed in`].join(' · ')}
        {signedIn && premium ? (
          <Box
            component="span"
            sx={{ color: 'var(--mortar-accent-ink)', '&::before': { content: '" · "' } }}
          >
            {t`Premium`}
          </Box>
        ) : null}
      </Box>
      {signedIn ? (
        <TitleMenuItem
          dim={true}
          icon={<LogOut size={ICON_PX} />}
          label={t`Sign out of ${{ source: sourceLabel('nexus') }}`}
          onClick={() => {
            close()
            SignOut().catch(reportUnexpected)
          }}
        />
      ) : (
        <TitleMenuItem
          icon={<LogIn size={ICON_PX} />}
          label={t`Sign in to ${{ source: sourceLabel('nexus') }}…`}
          onClick={() => {
            close()
            openSettings('accounts')
          }}
        />
      )}
    </>
  )
}

// Ways to reach the project (its Discord, bug reports and suggestions, about, its source), then the startup-profile
// restart, which is troubleshooting.
function ProjectLinks({ close }: { close: () => void }) {
  const { t } = useLingui()
  const open = (url: string) => () => {
    close()
    openPage(url)
  }
  return (
    <>
      <TitleMenuItem
        icon={<DiscordLogo size={ICON_PX} />}
        label={t`Ask on ${{ app: sourceLabel('discord') }}`}
        onClick={open(`${GO}help/`)}
      />
      <TitleMenuItem
        icon={<Bug size={ICON_PX} />}
        label={t`Report a bug…`}
        onClick={() => {
          close()
          reportBug(routeGame(useNav.getState().route) ?? '')
        }}
      />
      <TitleMenuItem
        icon={<Lightbulb size={ICON_PX} />}
        label={t`Suggest a feature…`}
        onClick={() => {
          close()
          suggestFeature()
        }}
      />
      <TitleMenuItem
        icon={<Info size={ICON_PX} />}
        label={t`About Mortar`}
        onClick={() => {
          close()
          openSettings('about')
        }}
      />
      <TitleMenuItem
        icon={<Code2 size={ICON_PX} />}
        label={t`Source code`}
        onClick={open(SOURCE)}
      />
      <TitleMenuItem
        icon={<Timer size={ICON_PX} />}
        label={t`Restart and record startup profile`}
        onClick={() => {
          close()
          RestartProfilingStartup().catch(reportError(t`Could not restart Mortar`))
        }}
      />
    </>
  )
}

// Its own component: only this item follows the open game, so a navigation leaves the other items alone.
function GetHelp() {
  const { t } = useLingui()
  const game = useNav((s) => routeGame(s.route) ?? '')
  const helpKeys = useShortcutHint('help')
  return (
    <TitleMenuItem
      icon={<LifeBuoy size={ICON_PX} />}
      label={t`Get help`}
      hint={helpKeys}
      disabled={game === ''}
      onClick={() => {
        closeTitleMenu()
        useTab.getState().setTab('console')
        useConsole.getState().setHelping(true)
      }}
    />
  )
}

// The menu's items, memoised apart from its button: opening the menu changes only the button and the menu around them.
const AppMenuItems = memo(function AppMenuItemsView() {
  const { t } = useLingui()
  const close = closeTitleMenu
  const updatesOffline = useUpdatesOfflineReason()
  const version = useMortarUpdate((s) => s.info?.version)
  const settingsKeys = useShortcutHint('open-settings')
  const downloadsKeys = useShortcutHint('downloads')
  const quit = (): void => {
    close()
    Application.Quit().catch(reportError(t`Could not quit Mortar`, quit))
  }
  return (
    <>
      <TitleMenuItem
        icon={<Download size={ICON_PX} />}
        label={t`Downloads`}
        hint={downloadsKeys}
        onClick={() => {
          close()
          useQueue.getState().setOpen(true)
        }}
      />
      <TitleMenuItem
        icon={<FolderOpen size={ICON_PX} />}
        label={t`Open data folder`}
        onClick={() => {
          close()
          OpenDataFolder().catch(reportUnexpected)
        }}
      />
      <TitleMenuItem
        icon={<Settings size={ICON_PX} />}
        label={t`Mortar settings…`}
        hint={settingsKeys}
        onClick={() => {
          close()
          openSettings()
        }}
      />
      <MenuRule />
      <MenuHeading>{t`Updates`}</MenuHeading>
      <Reason reason={updatesOffline}>
        <TitleMenuItem
          icon={<RefreshCw size={ICON_PX} />}
          label={t`Check mods and Mortar for updates`}
          disabled={updatesOffline !== ''}
          onClick={() => {
            close()
            checkForUpdates().catch(reportUnexpected)
          }}
        />
      </Reason>
      {version ? (
        <TitleMenuItem
          icon={<Sparkles size={ICON_PX} />}
          label={t`What is new in ${version}`}
          onClick={() => {
            close()
            showWhatsNew(version)
          }}
        />
      ) : null}
      <MenuRule />
      <MenuHeading>{t`Help`}</MenuHeading>
      <GetHelp />
      <ProjectLinks close={close} />
      <MenuRule />
      <NexusLines close={close} />
      <MenuRule />
      <TitleMenuItem icon={<LogOut size={ICON_PX} />} label={t`Quit Mortar`} onClick={quit} />
    </>
  )
})

// Memoised: the title bar re-renders on every navigation, and this one does not need to.
export const AppMenu = memo(function AppMenuView() {
  const { t } = useLingui()
  const { anchor, close, trigger } = useTitleMenu('app')
  useEffect(() => {
    useMortarUpdate.getState().load().catch(reportUnexpected)
  }, [])
  return (
    <>
      <ButtonBase
        aria-label={t`Mortar menu`}
        data-tour="app-menu"
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        {...trigger}
        sx={{
          '--wails-draggable': 'no-drag',
          gap: space.gap,
          height: space.control,
          px: '10px',
          borderRadius: '8px',
          fontSize: 'lead.fontSize',
          fontWeight: 700,
          fontFamily: 'inherit',
          color: 'inherit',
          '&:hover, &[aria-expanded="true"]': { bgcolor: 'var(--mortar-hairline-faint)' },
          [compact]: { px: '8px', '& .label': { display: 'none' } },
        }}
      >
        <Logo size={18} />
        <span className="label">{t`Mortar`}</span>
        <ChevronDown size={14} aria-hidden={true} className="label" />
      </ButtonBase>
      <TitleMenu anchorEl={anchor} onClose={close} label={t`Mortar`} width={280}>
        <AppMenuItems />
      </TitleMenu>
    </>
  )
})
