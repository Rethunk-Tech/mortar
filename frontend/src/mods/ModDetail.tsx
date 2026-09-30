import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
  IconButton,
  Link,
  Typography,
} from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { X } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type {
  Need,
  Relations,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type {
  Mod,
  ModState,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDetail } from './detail.ts'
import { modId, siblingsOf, sourceKind, updateFor } from './lookup.ts'
import { accent, heading, paper } from './paper.ts'
import { LetterTile, ModSwitch } from './parts.tsx'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

const WIDTH = 380
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
    disabled: t`Switched off`,
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

function Header({ mod, profile, pageUrl }: { mod: Mod; profile: Profile; pageUrl: string }) {
  const { t } = useLingui()
  const show = useDetail((s) => s.show)
  return (
    <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1.5, p: 2 }}>
      <LetterTile mod={mod} size={52} />
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography sx={{ fontSize: 16, fontWeight: 700, overflowWrap: 'anywhere' }}>
          {mod.name}
        </Typography>
        <Typography sx={{ ...text, color: 'text.secondary', overflowWrap: 'anywhere' }}>
          {pageUrl ? `${mod.author} · ` : mod.author}
          {pageUrl ? (
            <Link
              component="button"
              onClick={() => Browser.OpenURL(pageUrl).catch(reportUnexpected)}
              sx={{ ...text, verticalAlign: 'baseline' }}
            >
              {pageUrl.includes('github.com') ? t`GitHub page` : t`Nexus page`}
            </Link>
          ) : null}
        </Typography>
        <FormControlLabel
          sx={{ ml: 0, mt: 0.5 }}
          control={<ModSwitch mod={mod} />}
          label={<Typography sx={text}>{t`Enabled in ${profile.name}`}</Typography>}
        />
      </Box>
      <IconButton aria-label={t`Close details`} onClick={() => show(null)}>
        <X size={16} />
      </IconButton>
    </Box>
  )
}

function UpdateBanner({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const update = useUpdates((s) => updateFor(s.updates, mod))
  const setReviewing = useUpdates((s) => s.setReviewing)
  if (!update) {
    return null
  }
  return (
    <Box
      sx={{
        mx: 2,
        mb: 1,
        px: 1.5,
        py: 0.75,
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        borderRadius: '6px',
        bgcolor: accent.fill,
        border: '1px solid',
        borderColor: accent.line,
      }}
    >
      <Typography sx={{ flex: 1, ...text }}>
        {t`Update available: ${update.installed} → ${update.version}`}
      </Typography>
      <Button size="small" variant="contained" onClick={() => setReviewing(true)} sx={noWrap}>
        {t`Update`}
      </Button>
    </Box>
  )
}

type Confirming = 'rollback' | 'reset' | null

function Versions({
  mod,
  state,
  ask,
}: {
  mod: Mod
  state?: ModState | undefined
  ask: () => void
}) {
  const { t } = useLingui()
  return (
    <Section title={t`Versions`}>
      <Typography sx={{ ...row }}>{t`${mod.version} · in use`}</Typography>
      {state?.previousVersion ? (
        <Box sx={row}>
          <Typography sx={{ flex: 1, ...text }}>
            {t`${state.previousVersion} · kept for rollback`}
          </Typography>
          <Button size="small" variant="outlined" onClick={ask} sx={noWrap}>
            {t`Roll back`}
          </Button>
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
  const showFiles = useMods((s) => s.showFiles)
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
            <Button
              size="small"
              variant="outlined"
              onClick={() => showFiles(mod).catch(reportUnexpected)}
              sx={noWrap}
            >
              {t`Open`}
            </Button>
            <Button size="small" variant="outlined" onClick={ask} sx={noWrap}>
              {t`Reset`}
            </Button>
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
    <Dialog open={confirming !== null} onClose={onClose} slotProps={{ paper }}>
      <DialogTitle>
        {rolling ? t`Roll back ${mod.name}?` : t`Reset the settings of ${mod.name}?`}
      </DialogTitle>
      <DialogContent>
        <DialogContentText>
          {rolling
            ? t`${mod.name} goes back to ${previous}, keeping its settings. Your saves are backed up first.`
            : t`config.json is deleted, and the mod writes a fresh one with its own defaults the next time the game runs.`}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
        <Button
          color={rolling ? 'primary' : 'error'}
          onClick={() => {
            const run = rolling ? rollBack : resetConfig
            onClose()
            run(mod).catch(reportUnexpected)
          }}
        >
          {rolling ? t`Roll back` : t`Reset`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function Footer({ mod, bundled }: { mod: Mod; bundled: boolean }) {
  const { t } = useLingui()
  const showFiles = useMods((s) => s.showFiles)
  const askRemove = useMods((s) => s.askRemove)
  return (
    <Box sx={{ display: 'grid', gridTemplateColumns: bundled ? '1fr' : '1fr 1fr', gap: 1, p: 2 }}>
      <Button variant="outlined" onClick={() => showFiles(mod).catch(reportUnexpected)} sx={noWrap}>
        {t`Show files`}
      </Button>
      {bundled ? null : (
        <Button variant="outlined" color="error" onClick={() => askRemove(mod)} sx={noWrap}>
          {t`Remove from profile`}
        </Button>
      )}
    </Box>
  )
}

function Body({ mod, relations, state, ask }: BodyProps) {
  const { t } = useLingui()
  const others = siblingsOf(
    useMods((s) => s.mods),
    mod,
  )
  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        overflowY: 'auto',
        px: 2,
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      <Section title={t`Needs`}>
        {(relations?.needs ?? []).length > 0 ? (
          (relations?.needs ?? []).map((n) => <NeedRow key={n.uniqueId} need={n} />)
        ) : (
          <Typography sx={text}>{t`Nothing`}</Typography>
        )}
      </Section>
      {others.length > 0 ? (
        <Section title={t`In the same download`}>
          {others.map((o) => (
            <Typography key={o.uniqueId} sx={text}>
              {o.name}{' '}
              <Box component="span" sx={{ color: 'text.secondary' }}>
                {t`· updates and rolls back with this mod`}
              </Box>
            </Typography>
          ))}
        </Section>
      ) : null}
      <Versions mod={mod} state={state} ask={() => ask('rollback')} />
      <Settings mod={mod} state={state} ask={() => ask('reset')} />
      <Section title={t`Needed by`}>
        <Names
          names={(relations?.neededBy ?? []).map((d) => d.name)}
          none={t`Nothing in this profile`}
        />
      </Section>
    </Box>
  )
}

interface BodyProps {
  mod: Mod
  relations?: Relations | undefined
  state?: ModState | undefined
  ask: (what: Confirming) => void
}

function Panel({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t } = useLingui()
  const extras = useDetail((s) => s.extras)
  const loadExtras = useDetail((s) => s.loadExtras)
  const [confirming, setConfirming] = useState<Confirming>(null)
  // The mod's files change with the profile's updated time.
  const updated = String(profile.updated)
  useEffect(() => {
    if (updated) {
      loadExtras(mod).catch(reportUnexpected)
    }
  }, [mod, updated, loadExtras])
  const mine = extras?.id === modId(mod) ? extras : null
  return (
    <Box
      role="complementary"
      aria-label={t`Mod details`}
      sx={{
        position: 'absolute',
        top: 0,
        right: 0,
        bottom: 0,
        width: WIDTH,
        maxWidth: '100%',
        zIndex: 2,
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'rgba(36,36,44,0.98)',
        borderLeft: '1px solid rgba(255,255,255,0.12)',
      }}
    >
      <Header mod={mod} profile={profile} pageUrl={mine?.relations.pageUrl ?? ''} />
      <UpdateBanner mod={mod} />
      <Body mod={mod} relations={mine?.relations} state={mine?.state} ask={setConfirming} />
      <Footer mod={mod} bundled={sourceKind(profile, mod) === 'smapi'} />
      <Confirm
        mod={mod}
        confirming={confirming}
        previous={mine?.state.previousVersion ?? ''}
        onClose={() => setConfirming(null)}
      />
    </Box>
  )
}

export function ModDetail({ profile }: { profile: Profile }) {
  const mods = useMods((s) => s.mods)
  const detailId = useDetail((s) => s.detailId)
  const mod = mods.find((m) => modId(m) === detailId)
  return mod ? <Panel mod={mod} profile={profile} /> : null
}
