import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Link,
  Typography,
} from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import type {
  Need,
  Relations,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type {
  Mod,
  ModState,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { CompatDetail } from './CompatChip.tsx'
import { EditConfigButton } from './ConfigEditor.tsx'
import { useDetail } from './detail.ts'
import { LockedNote } from './LockedNote.tsx'
import { LockedReason } from './LockedReason.tsx'
import { entryOf, modId, siblingsOf } from './lookup.ts'
import { openPage } from './menu.ts'
import { NexusDetails } from './NexusDetails.tsx'
import { heading } from './paper.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const text = { fontSize: 13 } as const
const row = { display: 'flex', alignItems: 'center', gap: 1, minHeight: 30, ...text } as const
const noWrap = { whiteSpace: 'nowrap' } as const

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Typography sx={heading}>{title}</Typography>
      {children}
    </Box>
  )
}

function NeedRow({ need }: { need: Need }) {
  const { t } = useLingui()
  const states: Record<string, string> = {
    absent: t`Missing`,
    disabled: t`Off`,
    outdated: t`Needs ${need.minimumVersion} or newer, has ${need.installedVersion}`,
  }
  return (
    <Box sx={row}>
      <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, ...text }}>
        {need.required ? need.name : t`${need.name} (optional)`}
      </Typography>
      <Typography
        sx={{
          fontSize: 12,
          ...noWrap,
          color: need.state === 'ok' ? 'text.secondary' : 'warning.main',
        }}
      >
        {states[need.state] ?? t`Installed`}
      </Typography>
    </Box>
  )
}

function Names({ names, none }: { names: string[]; none: string }) {
  return names.length > 0 ? (
    names.map((n) => (
      <Typography key={n} sx={text}>
        {n}
      </Typography>
    ))
  ) : (
    <Typography sx={text}>{none}</Typography>
  )
}

type Confirming = 'rollback' | 'reset' | null

function Versions({
  mod,
  profile,
  state,
  ask,
}: {
  mod: Mod
  profile: Profile
  state?: ModState | undefined
  ask: () => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const entry = entryOf(profile, mod.key)
  return (
    <Section title={t`Versions`}>
      <Typography sx={{ ...row }}>{t`${mod.version} · in use`}</Typography>
      {entry?.pinned && entry.pinReason ? (
        <Typography
          sx={{ ...row, color: 'text.secondary' }}
        >{t`Pinned: ${entry.pinReason}`}</Typography>
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
  ask,
}: {
  mod: Mod
  state?: ModState | undefined
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
  const label = labels[state?.config ?? ''] ?? t`No config.json yet`
  const hasConfig = label !== t`No config.json yet`
  return (
    <Section title={t`Settings`}>
      <Box sx={row}>
        <Typography sx={{ flex: 1, ...text }}>{label}</Typography>
        {hasConfig ? (
          <>
            <EditConfigButton mod={mod} />
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

function Body({ mod, profile, relations, state, ask }: BodyProps) {
  const { t } = useLingui()
  const others = siblingsOf(
    useMods((s) => s.mods),
    mod,
  )
  return (
    <>
      <Section title={t`Needs`}>
        {(relations?.needs ?? []).length > 0 ? (
          (relations?.needs ?? []).map((n) => <NeedRow key={n.id} need={n} />)
        ) : (
          <Typography sx={text}>{t`Nothing`}</Typography>
        )}
      </Section>
      {others.length > 0 ? (
        <Section title={t`In the same download`}>
          {others.map((o) => (
            <Typography key={o.id} sx={text}>
              {o.name}{' '}
              <Box component="span" sx={{ color: 'text.secondary' }}>
                {t`· updates and rolls back with this mod`}
              </Box>
            </Typography>
          ))}
        </Section>
      ) : null}
      <Versions mod={mod} profile={profile} state={state} ask={() => ask('rollback')} />
      <Settings mod={mod} state={state} ask={() => ask('reset')} />
      <CompatDetail mod={mod} />
      <Section title={t`Needed by`}>
        <Names
          names={(relations?.neededBy ?? []).map((d) => d.name)}
          none={t`Nothing in this profile`}
        />
      </Section>
    </>
  )
}

interface BodyProps {
  mod: Mod
  profile: Profile
  relations?: Relations | undefined
  state?: ModState | undefined
  ask: (what: Confirming) => void
}

const isGitHubPage = (url: string): boolean => {
  try {
    const host = new URL(url).hostname
    return host === 'github.com' || host.endsWith('.github.com')
  } catch {
    return false
  }
}

function PageLink({ url }: { url: string }) {
  const { t } = useLingui()
  return url ? (
    <Link
      component="button"
      onClick={() => openPage(url)}
      sx={{ ...text, alignSelf: 'flex-start' }}
    >
      {isGitHubPage(url) ? t`GitHub page` : t`Nexus page`}
    </Link>
  ) : null
}

function Details({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t } = useLingui()
  const extras = useDetail((s) => s.extras)
  const loadExtras = useDetail((s) => s.loadExtras)
  const setOpen = useDetail((s) => s.setOpen)
  const [confirming, setConfirming] = useState<Confirming>(null)
  // The mod's files change with the profile's updated time.
  const updated = String(profile.updated)
  useEffect(() => {
    if (updated) {
      loadExtras(mod).catch(reportUnexpected)
    }
  }, [mod, updated, loadExtras])
  const mine = extras?.id === modId(mod) ? extras : null
  const source = (profile.entries ?? []).find((e) => e.key === mod.key)?.source
  return (
    <Dialog open={true} onClose={() => setOpen(false)} fullWidth={true} maxWidth="sm">
      <DialogTitle>{mod.name}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Box sx={{ mx: -2 }}>
          <LockedNote />
        </Box>
        <PageLink url={mine?.relations.pageUrl ?? ''} />
        {source?.kind === 'nexus' && source.modId ? (
          <NexusDetails mod={mod} modId={source.modId} fileId={source.fileId ?? 0} />
        ) : null}
        <Body
          mod={mod}
          profile={profile}
          relations={mine?.relations}
          state={mine?.state}
          ask={setConfirming}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={() => setOpen(false)}>{t`Close`}</Button>
      </DialogActions>
      <Confirm
        mod={mod}
        confirming={confirming}
        previous={mine?.state.previousVersion ?? ''}
        onClose={() => setConfirming(null)}
      />
    </Dialog>
  )
}

export function ModDetail({ profile }: { profile: Profile }) {
  const mods = useMods((s) => s.mods)
  const detailId = useDetail((s) => s.detailId)
  const open = useDetail((s) => s.open)
  const mod = mods.find((m) => modId(m) === detailId)
  return open && mod ? <Details mod={mod} profile={profile} /> : null
}
