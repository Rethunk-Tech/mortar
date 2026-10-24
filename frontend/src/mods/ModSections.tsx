import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { type ReactNode, useState } from 'react'
import type { ConfigFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/models.ts'
import type {
  Mod,
  ModState,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { Tooltip } from '../shell/Tooltip.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { ChangeVersionDialog } from './ChangeVersionDialog.tsx'
import { useDetail } from './detail.ts'
import { LockedReason } from './LockedReason.tsx'
import { entryOf, modId } from './lookup.ts'
import { ModUpdateControls } from './ModUpdateControls.tsx'
import { heading } from './paper.ts'
import { useMods } from './store.ts'
import { EditConfigButton } from './typedConfig/EditConfigButton.tsx'
import { useLocked } from './useLocked.ts'
import { targetOfSource } from './versionChoices.ts'

const text = { fontSize: 'base.fontSize' } as const
const row = {
  display: 'flex',
  alignItems: 'center',
  gap: space.gap,
  minHeight: 30,
  ...text,
} as const

export function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Typography sx={heading}>{title}</Typography>
      {children}
    </Box>
  )
}

type Confirming = 'rollback' | 'reset' | null

// The installed version with what changes it: another version, the pin, a rollback and the update channel. notices
// (an update on offer, a skipped one) sit under the version they are about.
export function ModVersion({
  mod,
  profile,
  notices,
}: {
  mod: Mod
  profile: Profile
  notices: ReactNode
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const extras = useDetail((s) => s.extras)
  const setPinned = useMods((s) => s.setPinned)
  const [confirming, setConfirming] = useState<Confirming>(null)
  const state = extras?.id === modId(mod) ? extras.state : undefined
  const entry = entryOf(profile, mod.key)
  const pinned = entry?.pinned && entry.pinReason ? entry.pinReason : ''
  const target = entry ? targetOfSource(entry.source) : null
  const [choosing, setChoosing] = useState(false)
  const game = useProfiles((s) => s.game?.id ?? '')
  return (
    <Section title={t`Version`}>
      <Typography sx={{ ...text, overflowWrap: 'anywhere' }}>{mod.version}</Typography>
      {notices}
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: space.gap }}>
        {target && entry ? (
          <LockedReason locked={locked}>
            <Button
              size="small"
              variant="outlined"
              disabled={locked}
              onClick={() => setChoosing(true)}
            >
              {t`Change version…`}
            </Button>
          </LockedReason>
        ) : null}
        <Tooltip
          describeChild={true}
          title={
            entry?.pinned
              ? t`Pinned: Mortar offers no update for this mod until you unpin it.`
              : t`Mortar stops offering updates for this mod and keeps the version you have.`
          }
        >
          <Button
            size="small"
            variant="outlined"
            onClick={() => setPinned(mod, !entry?.pinned).catch(reportUnexpected)}
          >
            {entry?.pinned ? t`Unpin version` : t`Pin this version`}
          </Button>
        </Tooltip>
      </Box>
      {pinned ? (
        <Typography sx={{ ...row, color: 'text.secondary' }}>{t`Pinned: ${pinned}`}</Typography>
      ) : null}
      {state?.previousVersion ? (
        <Box sx={row}>
          <Typography sx={{ flex: 1, ...text }}>
            {`${state.previousVersion} · ${t`kept for rollback`}`}
          </Typography>
          <LockedReason locked={locked}>
            <Button
              size="small"
              variant="outlined"
              disabled={locked}
              onClick={() => setConfirming('rollback')}
            >
              {t`Roll back`}
            </Button>
          </LockedReason>
        </Box>
      ) : null}
      <ModUpdateControls mod={mod} entry={entry} />
      {choosing && target && entry ? (
        <ChangeVersionDialog
          game={game}
          name={mod.name}
          target={target}
          entry={{
            key: entry.key,
            version: entry.source.version ?? mod.version,
            pinned: entry.pinned === true,
          }}
          onClose={() => setChoosing(false)}
        />
      ) : null}
      <Confirm
        mod={mod}
        confirming={confirming}
        previous={state?.previousVersion ?? ''}
        onClose={() => setConfirming(null)}
      />
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
      <Typography sx={text}>{label}</Typography>
      <Box
        sx={{ display: 'flex', flexWrap: 'wrap', gap: space.gap, '&:empty': { display: 'none' } }}
      >
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
      title={
        rolling
          ? t`Roll back ${{ name: mod.name }}?`
          : t`Reset the settings of ${{ name: mod.name }}?`
      }
      body={
        rolling
          ? t`${{ name: mod.name }} goes back to ${previous}, keeping its settings. Your saves are backed up first.`
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

// The mod's settings file and its reset.
export function ModSettings({ mod, profile }: { mod: Mod; profile: Profile }) {
  const extras = useDetail((s) => s.extras)
  const [confirming, setConfirming] = useState<Confirming>(null)
  const mine = extras?.id === modId(mod) ? extras : null
  return (
    <>
      <Settings
        mod={mod}
        state={mine?.state}
        configFiles={mine?.configFiles ?? []}
        isPackage={entryOf(profile, mod.key)?.package === true}
        ask={() => setConfirming('reset')}
      />
      <Confirm mod={mod} confirming={confirming} previous="" onClose={() => setConfirming(null)} />
    </>
  )
}
