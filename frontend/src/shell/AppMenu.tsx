import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { Application } from '@wailsio/runtime'
import {
  Bug,
  ChevronDown,
  Code2,
  FolderOpen,
  Info,
  LifeBuoy,
  LogIn,
  LogOut,
  RefreshCw,
  Settings,
  Sparkles,
} from 'lucide-react'
import { type ReactNode, useEffect } from 'react'
import { SignOut } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { OpenDataFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { Logo } from '../brand/Logo.tsx'
import { useConsole } from '../console/store.ts'
import { compact } from '../game/compact.ts'
import { useTab } from '../game/tab.ts'
import { openPage } from '../mods/menu.ts'
import { openSettings, routeGame, useNav } from '../nav/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { useMortarUpdate } from '../settings/updates.ts'
import { useShortcutHint } from '../settings/useShortcutHint.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { showWhatsNew } from '../updates/whatsNew.ts'
import { checkForUpdates } from './checkForUpdates.ts'
import { OneTip } from './OneTip.tsx'
import { useUpdatesOfflineReason } from './offlineText.ts'
import { reportBug } from './reportBug.ts'
import { MenuHeading, MenuRule, TitleMenu, TitleMenuItem } from './TitleMenu.tsx'
import { useTitleMenu } from './titleMenus.ts'

const ICON_PX = 18
const SOURCE = 'https://github.com/Rethunk-Tech/mortar'

// A disabled row cannot take hover, so its reason rides on a wrapper.
function Reason({ reason, children }: { reason: string; children: ReactNode }) {
  return reason === '' ? (
    children
  ) : (
    <OneTip title={reason} placement="left" describeChild={true}>
      <span>{children}</span>
    </OneTip>
  )
}

function NexusLines({ close }: { close: () => void }) {
  const { t } = useLingui()
  const { signedIn, name, premium } = useNexus()
  return (
    <>
      <Box sx={{ px: '14px', py: '6px', fontSize: 13 }}>
        {signedIn ? t`Nexus Mods · Signed in as ${name}` : t`Nexus Mods · Not signed in`}
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
          label={t`Sign out of Nexus`}
          onClick={() => {
            close()
            SignOut().catch(reportUnexpected)
          }}
        />
      ) : (
        <TitleMenuItem
          icon={<LogIn size={ICON_PX} />}
          label={t`Sign in to Nexus…`}
          onClick={() => {
            close()
            openSettings('accounts')
          }}
        />
      )}
    </>
  )
}

export function AppMenu() {
  const { t } = useLingui()
  const { anchor, close, trigger } = useTitleMenu('app')
  const updatesOffline = useUpdatesOfflineReason()
  const game = useNav((s) => routeGame(s.route) ?? '')
  const version = useMortarUpdate((s) => s.info?.version)
  const settingsKeys = useShortcutHint('open-settings')
  const helpKeys = useShortcutHint('help')
  useEffect(() => {
    useMortarUpdate
      .getState()
      .load()
      .catch(() => undefined)
  }, [])
  const quit = (): void => {
    close()
    Application.Quit().catch((e: unknown) =>
      toastError(t`Could not quit Mortar`, e, { retry: quit }),
    )
  }
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
          gap: '8px',
          height: 34,
          px: '10px',
          borderRadius: '8px',
          fontSize: 15,
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
            label={t`What's new in ${version}`}
            onClick={() => {
              close()
              showWhatsNew(version)
            }}
          />
        ) : null}
        <TitleMenuItem
          icon={<FolderOpen size={ICON_PX} />}
          label={t`Open data folder`}
          onClick={() => {
            close()
            OpenDataFolder().catch(reportUnexpected)
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
        <MenuRule />
        <MenuHeading>{t`Help`}</MenuHeading>
        <TitleMenuItem
          icon={<LifeBuoy size={ICON_PX} />}
          label={t`Get help`}
          hint={helpKeys}
          disabled={game === ''}
          onClick={() => {
            close()
            useTab.getState().setTab('console')
            useConsole.getState().setHelping(true)
          }}
        />
        <TitleMenuItem
          icon={<Bug size={ICON_PX} />}
          label={t`Report a bug…`}
          onClick={() => {
            close()
            reportBug(game)
          }}
        />
        <TitleMenuItem
          icon={<Code2 size={ICON_PX} />}
          label={t`Source code`}
          onClick={() => {
            close()
            openPage(SOURCE).catch(reportUnexpected)
          }}
        />
        <MenuRule />
        <NexusLines close={close} />
        <MenuRule />
        <TitleMenuItem icon={<LogOut size={ICON_PX} />} label={t`Quit Mortar`} onClick={quit} />
      </TitleMenu>
    </>
  )
}
