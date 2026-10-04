import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { ComponentType } from 'react'
import type { Drift } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  AdoptDriftFolder,
  ForgetDriftEntry,
  KeepDriftChanges,
  RemoveDriftFolder,
  RestoreDriftEntry,
  RevertDriftEntry,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { Problem } from './lookup.ts'
import { AssetFix } from './problemFix/AssetFix.tsx'
import { BrokenFix } from './problemFix/BrokenFix.tsx'
import { DismissedBrokenFix } from './problemFix/DismissedBrokenFix.tsx'
import { DuplicateFix } from './problemFix/DuplicateFix.tsx'
import { MissingFix } from './problemFix/MissingFix.tsx'
import { SettingFix } from './problemFix/SettingFix.tsx'
import { WinFix } from './problemFix/WinFix.tsx'
import type { WarningButton } from './problemFix/warningButton.tsx'
import { useWarningButton } from './problemFix/warningButton.tsx'
import { RunErrorButtons } from './runErrorFixButtons.tsx'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

interface KindProps<K extends Problem['kind']> {
  problem: Extract<Problem, { kind: K }>
  dismissedToken?: string | undefined
  button: WarningButton
}

function RunErrorKind({ problem, button }: KindProps<'runError'>) {
  return <RunErrorButtons runError={problem.runError} button={button} />
}

function DuplicateKind({ problem, button }: KindProps<'duplicate'>) {
  return <DuplicateFix problem={problem} button={button} />
}

function BrokenKind({ problem, dismissedToken, button }: KindProps<'broken'>) {
  if (dismissedToken !== undefined) {
    return <DismissedBrokenFix problem={problem} dismissedToken={dismissedToken} button={button} />
  }
  return <BrokenFix problem={problem} button={button} />
}

function SettingKind({ problem, dismissedToken }: KindProps<'setting'>) {
  return <SettingFix problem={problem} dismissedToken={dismissedToken} />
}

function AssetKind({ problem, dismissedToken }: KindProps<'asset'>) {
  return (
    <AssetFix
      problem={problem}
      dismissedToken={dismissedToken}
      win={(primary) => <WinFix problem={problem} primary={primary} />}
    />
  )
}

function MissingKind({ problem, dismissedToken, button }: KindProps<'missing'>) {
  return <MissingFix problem={problem} dismissedToken={dismissedToken} button={button} />
}

const problemFixes = {
  runError: RunErrorKind,
  duplicate: DuplicateKind,
  broken: BrokenKind,
  setting: SettingKind,
  asset: AssetKind,
  missing: MissingKind,
} satisfies { [K in Problem['kind']]: ComponentType<KindProps<K>> }

export function FixButton({
  problem,
  dismissedToken,
}: {
  problem: Problem
  dismissedToken?: string | undefined
}) {
  const button = useWarningButton()
  const Kind = problemFixes[problem.kind] as ComponentType<{
    problem: Problem
    dismissedToken?: string | undefined
    button: WarningButton
  }>
  return <Kind problem={problem} dismissedToken={dismissedToken} button={button} />
}

export function DriftButtons({ drift }: { drift: Drift }) {
  const { t } = useLingui()
  const load = useMods((s) => s.load)
  const loadProblems = useMods((s) => s.loadProblems)
  const replace = useProfiles((s) => s.replace)
  const locked = useLocked()
  const target = () => {
    const { game, openId } = useProfiles.getState()
    return game && openId ? { game: game.id, id: openId } : null
  }
  const run = (work: () => Promise<unknown>) => {
    work()
      .then(() => Promise.all([load(), loadProblems()]))
      .catch(reportUnexpected)
  }
  const button = (label: string, onClick: () => void) => (
    <Button
      size="small"
      variant="contained"
      color="warning"
      disabled={locked}
      onClick={onClick}
      sx={{ flexShrink: 0 }}
    >
      {label}
    </Button>
  )
  if (drift.kind === 'unknown') {
    return (
      <>
        {button(t`Adopt`, () =>
          run(async () => {
            const open = target()
            if (!open) {
              return
            }
            replace(await AdoptDriftFolder(open.game, open.id, drift.folder))
          }),
        )}
        {button(t`Remove`, () =>
          run(async () => {
            const open = target()
            if (!open) {
              return
            }
            await RemoveDriftFolder(open.game, open.id, drift.folder)
          }),
        )}
      </>
    )
  }
  if (drift.kind === 'deleted') {
    return (
      <>
        {button(t`Restore`, () =>
          run(async () => {
            const open = target()
            if (!open) {
              return
            }
            replace(await RestoreDriftEntry(open.game, open.id, drift.key))
          }),
        )}
        {button(t`Forget`, () =>
          run(async () => {
            const open = target()
            if (!open) {
              return
            }
            replace(await ForgetDriftEntry(open.game, open.id, drift.key))
          }),
        )}
      </>
    )
  }
  return (
    <>
      {button(t`Keep my changes`, () =>
        run(async () => {
          const open = target()
          if (!open) {
            return
          }
          await KeepDriftChanges(open.game, open.id, drift.key)
        }),
      )}
      {button(t`Revert`, () =>
        run(async () => {
          const open = target()
          if (!open) {
            return
          }
          replace(await RevertDriftEntry(open.game, open.id, drift.key))
        }),
      )}
    </>
  )
}
