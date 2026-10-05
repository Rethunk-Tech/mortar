import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Copy, Map as MapIcon, ShieldCheck } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Compat } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { ConflictEvidence } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { type SectionAction, SectionStrip } from '../shell/SectionStrip.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { AssetMapDialog } from './AssetMapDialog.tsx'
import { CleanupSection } from './CleanupSection.tsx'
import { CompatSection } from './CompatSection.tsx'
import { compatReportChunks } from './compatChip.ts'
import { LockedNote } from './LockedNote.tsx'
import { ProblemRows } from './ProblemRows.tsx'
import { problemSections, type Row } from './problemGroups.ts'
import { formatProblemReport, whyKeysOf } from './problemReport.ts'
import { chosenSection, type SectionTab, useProblemSection } from './problemSection.ts'
import { ERROR_SECTIONS, isDismissedRow, useRowText, useSectionTitle } from './problemText.ts'
import { useRedundantRows } from './redundantReason.ts'
import { CheckTimings, SlowStartupSection } from './SlowStartupSection.tsx'
import { useSlowStartups } from './slowStartups.ts'
import { useMods } from './store.ts'
import { useLoadProblemsOnFocus } from './useLoadProblemsOnFocus.ts'

const PROBLEM_SKELETON_ROWS = 5
const PROBLEM_SKELETON_HEIGHT = 56

// useOpenProblems is the open profile's problems, or null while they load, never the profile shown before.
function useOpenProblems() {
  const openId = useProfiles((s) => s.openId)
  return useMods((s) => (s.problemsFor === openId ? s.problems : null))
}

function dismissCosmetic(
  sections: ReturnType<typeof problemSections>,
  dismissAsset: ReturnType<typeof useMods.getState>['dismissAsset'],
) {
  const cosmetic = sections.find((section) => section.id === 'cosmetic')
  for (const row of cosmetic?.rows ?? []) {
    if (!isDismissedRow(row) && row.kind === 'asset') {
      for (const asset of [row.asset, ...(row.siblings ?? [])]) {
        dismissAsset(asset).catch(reportUnexpected)
      }
    }
  }
}

function OfflineChecksNote() {
  const { t } = useLingui()
  return (
    <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
      {t`Some checks could not run without a connection, so more problems may show up later.`}
    </Typography>
  )
}

type ProblemTab = SectionTab & { body: ReactNode; action?: SectionAction }

// The tabs of the section switcher: a segment for each section that has anything in it.
function useProblemTabs({
  sections,
  compat,
  redundant,
  cleanup,
  cosmeticConflicts,
  sectionExtras,
}: {
  sections: ReturnType<typeof problemSections>
  compat: Compat[]
  redundant: ReturnType<ReturnType<typeof useRedundantRows>>
  cleanup: Parameters<typeof CleanupSection>[0]['cleanup']
  cosmeticConflicts: string
  sectionExtras: (section: ReturnType<typeof problemSections>[number]) => { action?: SectionAction }
}): ProblemTab[] {
  const { t } = useLingui()
  const sectionTitle = useSectionTitle()
  const slowCount = useSlowStartups().length
  return [
    ...sections
      .filter((s) => !(s.id === 'cosmetic' && cosmeticConflicts === 'hidden'))
      .map((s) => ({
        id: s.id,
        label: sectionTitle(s.id),
        count: s.rows.length,
        errors: ERROR_SECTIONS.has(s.id),
        body: <ProblemRows rows={s.rows} />,
        ...sectionExtras(s),
      })),
    {
      id: 'compat',
      label: t`Compatibility`,
      count: compat.length,
      errors: false,
      body: <CompatSection rows={compat} />,
    },
    {
      id: 'slow',
      label: t`Slow startup`,
      count: slowCount,
      errors: false,
      body: <SlowStartupSection />,
    },
    {
      id: 'redundant',
      label: t`Redundant`,
      count: redundant.length,
      errors: false,
      body: (
        <CleanupSection
          cleanup={redundant}
          removeAll={redundant.every((item) => item.choices === undefined)}
        />
      ),
    },
    {
      id: 'cleanup',
      label: t`Cleanup`,
      count: cleanup.length,
      errors: false,
      body: <CleanupSection cleanup={cleanup} />,
    },
  ].filter((tab) => tab.count > 0)
}

function ProblemsContent({ result }: { result: NonNullable<ReturnType<typeof useOpenProblems>> }) {
  const { t } = useLingui()
  const dismissAsset = useMods((s) => s.dismissAsset)
  const [confirmDismissCosmetic, setConfirmDismissCosmetic] = useState(false)
  const [addingAll, runAddAll] = usePending()
  const cosmeticConflicts = useSettings((s) => gamePrefs(s).cosmeticConflicts)
  const redundantRows = useRedundantRows()

  const sections = problemSections(result)
  const cleanup = result.cleanup ?? []
  const compat = result.compat ?? []
  const redundant = redundantRows(result.redundant ?? [])
  const installable =
    sections
      .find((section) => section.id === 'missing')
      ?.rows.filter(
        (row): row is Extract<Row, { kind: 'missing' }> =>
          !isDismissedRow(row) &&
          row.kind === 'missing' &&
          row.missing.listed &&
          row.missing.where !== null,
      ) ?? []
  const empty =
    sections.filter((s) => s.id !== 'dismissed').length === 0 &&
    cleanup.length + redundant.length + compat.length === 0 &&
    !result.unknown

  const sectionExtras = (section: (typeof sections)[number]): { action?: SectionAction } => {
    if (section.id === 'missing' && installable.length > 0) {
      return {
        action: {
          label: t`Add all ${installable.length}`,
          disabled: addingAll,
          onClick: () => {
            const wants = installable.flatMap(({ missing }) => {
              const want = missing.where ? refWant(missing.where, 'dependency') : null
              return want ? [want] : []
            })
            runAddAll(() => download(wants))
          },
        },
      }
    }
    if (section.id === 'cosmetic') {
      return {
        action: {
          label: t`Dismiss all`,
          onClick: () => {
            setConfirmDismissCosmetic(true)
          },
        },
      }
    }
    return {}
  }

  const openId = useProfiles((s) => s.openId)
  const remembered = useProblemSection((s) => s.byProfile[openId])
  const choose = useProblemSection((s) => s.choose)
  const tabs = useProblemTabs({
    sections,
    compat,
    redundant,
    cleanup,
    cosmeticConflicts,
    sectionExtras,
  })
  const current = chosenSection(tabs, remembered)
  const shown = tabs.find((tab) => tab.id === current)

  return (
    <>
      {tabs.length > 0 ? (
        <SectionStrip
          label={t`Problem sections`}
          tabs={tabs}
          current={current}
          onChoose={(id) => choose(openId, id)}
          action={shown?.action}
        />
      ) : null}
      <Box
        sx={{
          flexShrink: 0,
          px: 2,
          pt: 1.5,
          pb: 1.5,
          display: 'flex',
          flexDirection: 'column',
          gap: 1.5,
        }}
      >
        <Box sx={{ mx: -2, mb: -1 }}>
          <LockedNote />
        </Box>
        {empty ? (
          <EmptyState
            icon={<ShieldCheck size={40} aria-hidden={true} />}
            title={t`No problems found`}
          >
            {t`Every mod has what it needs and nothing clashes.`}
          </EmptyState>
        ) : null}
        {shown?.body}
        <ConfirmDialog
          open={confirmDismissCosmetic}
          title={t`Dismiss all harmless overlaps?`}
          body={t`They move to Dismissed. Restore them from that section.`}
          confirmLabel={t`Dismiss all`}
          onCancel={() => setConfirmDismissCosmetic(false)}
          onConfirm={() => {
            setConfirmDismissCosmetic(false)
            dismissCosmetic(sections, dismissAsset)
          }}
        />
        {result.unknown ? <OfflineChecksNote /> : null}
        <CheckTimings timings={result.timings ?? []} />
      </Box>
    </>
  )
}

// ProblemActions sits in the profile's tab row while the Problems tab is open, like the Console's log actions.
export function ProblemActions() {
  const { t } = useLingui()
  const result = useOpenProblems()
  const [mapOpen, setMapOpen] = useState(false)
  const sectionTitle = useSectionTitle()
  const rowText = useRowText()
  const sections = result === null ? [] : problemSections(result)
  const cleanup = result?.cleanup ?? []
  const compat = result?.compat ?? []
  const redundantRows = useRedundantRows()
  const redundant = redundantRows(result?.redundant ?? [])
  const harmlessCount = (result?.assetConflicts ?? []).filter((asset) => asset.cosmetic).length
  const nothing =
    result === null ||
    (sections.length === 0 &&
      cleanup.length + redundant.length + compat.length === 0 &&
      harmlessCount === 0)
  return (
    <>
      <IconAction
        label={t`Asset map`}
        icon={<MapIcon size={16} />}
        onClick={() => setMapOpen(true)}
      />
      <AssetMapDialog open={mapOpen} onClose={() => setMapOpen(false)} />
      <IconAction
        label={t`Copy report`}
        icon={<Copy size={16} />}
        disabled={nothing}
        disabledTitle={t`No problems to copy.`}
        onClick={() => {
          const { game, openId } = useProfiles.getState()
          const assets = sections.flatMap((section) =>
            section.rows.flatMap((entry) => {
              const row = isDismissedRow(entry) ? entry.row : entry
              return row.kind === 'asset' ? [row.asset] : []
            }),
          )
          Promise.all(
            assets.map((asset) =>
              game && openId
                ? ConflictEvidence(game.id, openId, asset.kind, asset.target)
                : Promise.resolve([]),
            ),
          )
            .then((rows) => {
              const evidenceOf = new Map(assets.map((asset, i) => [asset, rows[i]]))
              const text = formatProblemReport(
                [
                  ...sections.map((section) => ({
                    title: sectionTitle(section.id),
                    count: section.rows.length,
                    whyKeys: section.rows.flatMap((entry) => {
                      const row = isDismissedRow(entry) ? entry.row : entry
                      return row.kind === 'asset' ? whyKeysOf(evidenceOf.get(row.asset)) : []
                    }),
                    lines: section.rows.map((entry) => {
                      const row = isDismissedRow(entry) ? entry.row : entry
                      const { text: sentence, note } = rowText(row)
                      return note === '' ? sentence : `${sentence} ${note}`
                    }),
                  })),
                  ...(cleanup.length === 0
                    ? []
                    : [
                        {
                          title: t`Cleanup`,
                          count: cleanup.length,
                          lines: cleanup.map((item) => {
                            const who = item.name.trim() === '' ? t`Unknown mod` : item.name
                            const reason = item.reason || t`Not needed by any enabled mod`
                            return `${who}: ${reason}`
                          }),
                        },
                      ]),
                  ...(redundant.length === 0
                    ? []
                    : [
                        {
                          title: t`Redundant`,
                          count: redundant.length,
                          lines: redundant.map(
                            (item) => item.text ?? `${item.name}: ${item.reason}`,
                          ),
                        },
                      ]),
                  ...compatReportChunks(compat, t`Compatibility`),
                ],
                t`Harmless`,
                harmlessCount,
              )
              return Clipboard.SetText(text)
            })
            .then(
              () => useToasts.getState().push({ kind: 'success', title: t`Report copied` }),
              reportUnexpected,
            )
        }}
      />
    </>
  )
}

export function ProblemsTab() {
  const { t } = useLingui()
  const result = useOpenProblems()
  useLoadProblemsOnFocus()
  if (result === null) {
    return (
      <SkeletonRows
        label={t`Checking the mods for problems…`}
        count={PROBLEM_SKELETON_ROWS}
        height={PROBLEM_SKELETON_HEIGHT}
        sx={{ px: 2, py: 1.5 }}
      />
    )
  }
  return <ProblemsContent result={result} />
}
