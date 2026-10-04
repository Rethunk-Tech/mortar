import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { type ComponentType, useState } from 'react'
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
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import type { Problem } from './lookup.ts'
import { AssetFix } from './problemFix/AssetFix.tsx'
import { BrokenFix } from './problemFix/BrokenFix.tsx'
import { DuplicateFix } from './problemFix/DuplicateFix.tsx'
import { MissingFix } from './problemFix/MissingFix.tsx'
import { SettingFix } from './problemFix/SettingFix.tsx'
import { WinFix } from './problemFix/WinFix.tsx'
import type { WarningButton } from './problemFix/warningButton.tsx'
import { useWarningButton } from './problemFix/warningButton.tsx'
import { RunErrorButtons } from './runErrorFixButtons.tsx'
import { useMods } from './store.ts'
import { openTarget } from './storeView.ts'
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
  return <BrokenFix problem={problem} dismissedToken={dismissedToken} button={button} />
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
  const replace = useProfiles((s) => s.replace)
  const locked = useLocked()
  const run = (work: () => Promise<unknown>) => {
    work()
      .then(() => load())
      .catch(reportUnexpected)
  }
  const [confirm, setConfirm] = useState<{
    title: string
    body: string
    label: string
    act: () => void
  } | null>(null)
  // The keeping action is the main one; the one that discards the user's files is secondary and asks first.
  const button = (label: string, onClick: () => void, discard = false) => (
    <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
      <Button
        size="small"
        variant={discard ? 'outlined' : 'contained'}
        color="warning"
        disabled={locked}
        onClick={onClick}
        sx={{ flexShrink: 0 }}
      >
        {label}
      </Button>
    </DisabledReason>
  )
  const confirmDialog = confirm ? (
    <ConfirmDialog
      open={true}
      title={confirm.title}
      body={confirm.body}
      confirmLabel={confirm.label}
      color="error"
      onCancel={() => setConfirm(null)}
      onConfirm={() => {
        setConfirm(null)
        confirm.act()
      }}
    />
  ) : null
  if (drift.kind === 'unknown') {
    return (
      <>
        {button(t`Adopt`, () =>
          run(async () => {
            const open = openTarget()
            if (!open) {
              return
            }
            replace(await AdoptDriftFolder(open.game, open.id, drift.folder))
          }),
        )}
        {button(
          t`Remove`,
          () =>
            setConfirm({
              title: t`Remove ${drift.folder}?`,
              body: t`Mortar did not install this folder. It moves to Mortar's trash.`,
              label: t`Remove`,
              act: () =>
                run(async () => {
                  const open = openTarget()
                  if (!open) {
                    return
                  }
                  await RemoveDriftFolder(open.game, open.id, drift.folder)
                }),
            }),
          true,
        )}
        {confirmDialog}
      </>
    )
  }
  if (drift.kind === 'deleted') {
    return (
      <>
        {button(t`Restore`, () =>
          run(async () => {
            const open = openTarget()
            if (!open) {
              return
            }
            replace(await RestoreDriftEntry(open.game, open.id, drift.key))
          }),
        )}
        {button(t`Forget`, () =>
          run(async () => {
            const open = openTarget()
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
          const open = openTarget()
          if (!open) {
            return
          }
          await KeepDriftChanges(open.game, open.id, drift.key)
        }),
      )}
      {button(
        t`Revert`,
        () =>
          setConfirm({
            title: t`Revert ${drift.key}?`,
            body: t`Your edits to its files are replaced with the installed copy.`,
            label: t`Revert`,
            act: () =>
              run(async () => {
                const open = openTarget()
                if (!open) {
                  return
                }
                replace(await RevertDriftEntry(open.game, open.id, drift.key))
              }),
          }),
        true,
      )}
      {confirmDialog}
    </>
  )
}
