import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { ReactNode, Ref } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { space } from '../theme/density.ts'
import { useDuration, useSmapiStartup } from './startupHooks.ts'
import { type Finding, type PhaseId, phaseSegments } from './startupView.ts'

// Each phase's colour in the stacked bar and its legend.
const PHASE_COLORS: Record<PhaseId, string> = {
  smapi: 'text.disabled',
  entry: 'primary.main',
  content: 'info.main',
  firstTicks: 'warning.main',
  intro: 'success.main',
}

function Section({
  title,
  anchor,
  children,
}: {
  title: string
  anchor?: Ref<HTMLElement>
  children: ReactNode
}) {
  return (
    <Box
      ref={anchor}
      component="section"
      sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}
    >
      <Typography component="h2" sx={{ fontSize: 18, fontWeight: 600 }}>
        {title}
      </Typography>
      {children}
    </Box>
  )
}

function FindingLine({ finding, onAct }: { finding: Finding; onAct: (finding: Finding) => void }) {
  const { t } = useLingui()
  const duration = useDuration()
  let text: string
  let action: string
  switch (finding.kind) {
    case 'heavy': {
      const { name } = finding.mod
      const [own, total] = [duration(finding.ms), duration(finding.titleMs)]
      text =
        finding.packs > 0
          ? t`${name} takes ${own} of your ${total} start; ${plural(finding.packs, { one: '# content pack causes', other: '# content packs cause' })} most of it.`
          : t`${name} takes ${own} of your ${total} start.`
      action = finding.packs > 0 ? t`Show packs` : t`Show mod`
      break
    }
    case 'slower': {
      const by = duration(finding.ms)
      text = t`This start was ${by} slower than the one before.`
      action = t`Compare`
      break
    }
    default: {
      const ms = duration(finding.ms)
      text = t`SMAPI loads mods for ${ms}.`
      action = t`Show slowest mods`
    }
  }
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.pad,
        px: space.pad,
        py: 0.75,
        borderRadius: '8px',
        bgcolor: 'var(--mortar-overlay-30)',
        border: '1px solid var(--mortar-hairline-12)',
      }}
    >
      <Typography sx={{ flex: 1, minWidth: 0 }}>{text}</Typography>
      <Button size="small" variant="outlined" onClick={() => onAct(finding)}>
        {action}
      </Button>
    </Box>
  )
}

// Shown only when something stands out; the Startup card says so when nothing does.
function Findings({ findings, onAct }: { findings: Finding[]; onAct: (finding: Finding) => void }) {
  const { t } = useLingui()
  if (findings.length === 0) {
    return null
  }
  return (
    <Section title={t`What to do`}>
      {findings.map((f) => (
        <FindingLine key={f.kind} finding={f} onAct={onAct} />
      ))}
    </Section>
  )
}

/** A phase narrower than this share of the bar is named in its tooltip only. */
const LABELLED_SHARE = 0.12

// The start as one bar of phases, each sized by its time and named inside when it is wide enough.
function Phases({ report }: { report: StartupReport }) {
  const { t } = useLingui()
  const duration = useDuration()
  const smapi = useSmapiStartup()
  const labels: Record<PhaseId, string> = smapi
    ? {
        smapi: t`SMAPI loads mods`,
        entry: t`Mods start`,
        content: t`Game content`,
        firstTicks: t`First updates`,
        intro: t`Title intro`,
      }
    : {
        smapi: t`Unity starts`,
        entry: t`BepInEx patches`,
        content: t`Plugins load`,
        firstTicks: t`First scene`,
        intro: t`To the main menu`,
      }
  const help: Record<PhaseId, string> = smapi
    ? {
        smapi: t`From launch until SMAPI has loaded every mod's code`,
        entry: t`Each mod's Entry method, run one after another`,
        content: t`The game loads its own content`,
        firstTicks: t`The game's first frames; mods doing setup work in update events show up here`,
        intro: t`The title screen's intro animation (Mortar skips it on measured launches)`,
      }
    : {
        smapi: t`From launch until BepInEx's preloader runs`,
        entry: t`BepInEx's preloader patches the game, then Unity starts it`,
        content: t`Each plugin's constructor and Awake, run one after another`,
        firstTicks: t`Until the game's first scene has loaded`,
        intro: t`From the first scene to the main menu. Plugins' work in Start, coroutines and scene hooks lands here, not on the plugin, and so does any wait for a click, such as Lethal Company's Online or LAN choice.`,
      }
  const segments = phaseSegments(report.phases)
  const total = segments.reduce((n, s) => n + s.ms, 0)
  return (
    <Box
      role="img"
      aria-label={segments.map((s) => `${labels[s.id]} ${duration(s.ms)}`).join(', ')}
      sx={{ display: 'flex', height: 30, borderRadius: '6px', overflow: 'hidden', gap: '2px' }}
    >
      {segments.map((s) => (
        <Tooltip key={s.id} title={`${labels[s.id]}: ${duration(s.ms)}. ${help[s.id]}`}>
          <Box
            sx={{
              flexGrow: s.ms,
              display: 'flex',
              alignItems: 'center',
              px: 1,
              minWidth: 0,
              overflow: 'hidden',
              whiteSpace: 'nowrap',
              fontSize: 12,
              fontWeight: 600,
              color: 'background.default',
              bgcolor: PHASE_COLORS[s.id],
            }}
          >
            {total > 0 && s.ms / total >= LABELLED_SHARE
              ? `${labels[s.id]} ${duration(s.ms)}`
              : null}
          </Box>
        </Tooltip>
      ))}
    </Box>
  )
}

export { Findings, Phases, Section }
