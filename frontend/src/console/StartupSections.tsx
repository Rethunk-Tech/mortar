import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Tooltip, Typography } from '@mui/material'
import type { ReactNode, Ref } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { useDuration, useSmapiStartup } from './startupHooks.ts'
import { type Finding, type PhaseId, phaseSegments } from './startupView.ts'

const PCT = 100

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
    <Box ref={anchor} component="section" sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Typography component="h3" sx={{ fontSize: 18, fontWeight: 600 }}>
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
        gap: 2,
        px: 1.75,
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

function Findings({ findings, onAct }: { findings: Finding[]; onAct: (finding: Finding) => void }) {
  const { t } = useLingui()
  return (
    <Section title={t`What to do`}>
      {findings.length === 0 ? (
        <Typography
          sx={{ color: 'text.secondary' }}
        >{t`Nothing stands out in this start.`}</Typography>
      ) : (
        findings.map((f) => <FindingLine key={f.kind} finding={f} onAct={onAct} />)
      )}
    </Section>
  )
}

// One row per phase: its name, time and a thin bar sized against the whole start, in the primary colour.
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
    <Section title={t`Where the time goes`}>
      {segments.map((s) => (
        <Tooltip key={s.id} title={help[s.id]}>
          <Box
            tabIndex={0}
            sx={{
              display: 'grid',
              gridTemplateColumns: 'minmax(120px, 200px) 80px 1fr',
              alignItems: 'center',
              columnGap: 2,
              py: 0.5,
              borderBottom: '1px solid var(--mortar-hairline-12)',
            }}
          >
            <span>{labels[s.id]}</span>
            <Box component="span" sx={{ textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>
              {duration(s.ms)}
            </Box>
            <Box sx={{ height: 4, borderRadius: 2, bgcolor: 'var(--mortar-hairline-12)' }}>
              <Box
                sx={{
                  height: '100%',
                  borderRadius: 2,
                  bgcolor: 'primary.main',
                  width: `${(s.ms / total) * PCT}%`,
                }}
              />
            </Box>
          </Box>
        </Tooltip>
      ))}
    </Section>
  )
}

export { Findings, Phases, Section }
