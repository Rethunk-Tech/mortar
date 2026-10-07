import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { type ReactNode, useState } from 'react'
import type { ConfigFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/models.ts'
import type {
  Mod,
  ModState,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useDetail } from './detail.ts'
import { LockedReason } from './LockedReason.tsx'
import { entryOf, modId } from './lookup.ts'
import { heading } from './paper.ts'
import { useMods } from './store.ts'
import { EditConfigButton } from './typedConfig/EditConfigButton.tsx'
import { useLocked } from './useLocked.ts'

const text = { fontSize: 13 } as const
const row = { display: 'flex', alignItems: 'center', gap: 1, minHeight: 30, ...text } as const

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Typography sx={heading}>{title}</Typography>
      {children}
    </Box>
  )
}

type Confirming = 'rollback' | 'reset' | null

// The pin's reason and the earlier version kept for a rollback; the current version is the panel's own Version field.
function Versions({
  mod,
  profile,
  state,
  ask,
}: {
  mod: Mod
  profile: Profile
  state: ModState | undefined
  ask: () => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const entry = entryOf(profile, mod.key)
  const pinned = entry?.pinned && entry.pinReason ? entry.pinReason : ''
  if (!(pinned || state?.previousVersion)) {
    return null
  }
  return (
    <Section title={t`Versions`}>
      {pinned ? (
        <Typography sx={{ ...row, color: 'text.secondary' }}>{t`Pinned: ${pinned}`}</Typography>
      ) : null}
      {state?.previousVersion ? (
        <Box sx={row}>
          <Typography sx={{ flex: 1, ...text }}>
            {t`${state.previousVersion} · kept for rollback`}
          </Typography>
          <LockedReason locked={locked}>
            <Button size="small" variant="outlined" disabled={locked} onClick={ask}>
              {t`Roll back`}
            </Button>
          </LockedReason>
        </Box>
      ) : null}
    </Section>
  )
}

function Settings({
  mod,
  state,
  configFiles,
  isPackage,
  ask,
}: {
  mod: Mod
  state?: ModState | undefined
  configFiles: ConfigFile[]
  isPackage: boolean
  ask: () => void
}) {
  const { t } = useLingui()
  const openConfig = useMods((s) => s.openConfig)
  const locked = useLocked()
  const labels: Record<string, string> = {
    default: t`config.json, as the mod ships it`,
    changed: t`config.json, changed from the mod's own`,
    generated: t`config.json, made by the mod`,
  }
  const file = labels[state?.config ?? '']
  const gmcm = state?.gmcm === true
  const cfgs = configFiles.filter((f) => f.format === 'bepinex').map((f) => f.label || f.name)
  let label = file ?? t`No config.json yet`
  if (!file && gmcm) {
    label = t`Settings from the mod's in-game menu`
  }
  // A package's plugins keep .cfg files that BepInEx writes on the first run with the plugin, never a config.json.
  if (isPackage) {
    label = cfgs.length > 0 ? cfgs.join(', ') : t`No config file yet`
  }
  return (
    <Section title={t`Settings`}>
      <Box sx={row}>
        <Typography sx={{ flex: 1, ...text }}>{label}</Typography>
        {configFiles.length > 0 ? <EditConfigButton mod={mod} /> : null}
        {file ? (
          <>
            <Button
              size="small"
              variant="outlined"
              onClick={() => openConfig(mod).catch(reportUnexpected)}
            >
              {t`Open`}
            </Button>
            <LockedReason locked={locked}>
              <Button size="small" variant="outlined" disabled={locked} onClick={ask}>
                {t`Reset`}
              </Button>
            </LockedReason>
          </>
        ) : null}
      </Box>
    </Section>
  )
}

function Confirm({
  mod,
  confirming,
  previous,
  onClose,
}: {
  mod: Mod
  confirming: Confirming
  previous: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const rollBack = useDetail((s) => s.rollBack)
  const resetConfig = useDetail((s) => s.resetConfig)
  const rolling = confirming === 'rollback'
  return (
    <ConfirmDialog
      open={confirming !== null}
      title={rolling ? t`Roll back ${mod.name}?` : t`Reset the settings of ${mod.name}?`}
      body={
        rolling
          ? t`${mod.name} goes back to ${previous}, keeping its settings. Your saves are backed up first.`
          : t`config.json is deleted, and the mod writes a fresh one with its own defaults the next time the game runs.`
      }
      confirmLabel={rolling ? t`Roll back` : t`Reset`}
      color={rolling ? 'primary' : 'error'}
      onCancel={onClose}
      onConfirm={() => {
        const run = rolling ? rollBack : resetConfig
        onClose()
        run(mod).catch(reportUnexpected)
      }}
    />
  )
}

// What the details panel adds for a mod: its earlier version, its settings file and whether either can be reset.
export function ModVersionsAndSettings({ mod, profile }: { mod: Mod; profile: Profile }) {
  const extras = useDetail((s) => s.extras)
  const [confirming, setConfirming] = useState<Confirming>(null)
  const mine = extras?.id === modId(mod) ? extras : null
  return (
    <>
      <Versions
        mod={mod}
        profile={profile}
        state={mine?.state}
        ask={() => setConfirming('rollback')}
      />
      <Settings
        mod={mod}
        state={mine?.state}
        configFiles={mine?.configFiles ?? []}
        isPackage={entryOf(profile, mod.key)?.package === true}
        ask={() => setConfirming('reset')}
      />
      <Confirm
        mod={mod}
        confirming={confirming}
        previous={mine?.state.previousVersion ?? ''}
        onClose={() => setConfirming(null)}
      />
    </>
  )
}
