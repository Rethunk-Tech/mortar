import { useLingui } from '@lingui/react/macro'
import { Box, Button, Tooltip, Typography } from '@mui/material'
import { Copy, Map as MapIcon, Plus, ShieldCheck } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Compat } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { ConflictEvidence } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { findCrashCause, useBisectBlockText } from '../commandPalette/crashBisect.ts'
import { PageActions } from '../game/PageActions.tsx'
import { useOpenOverride } from '../profiles/openOverrides.ts'
import { useProfileLoader, useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { copyText } from '../share/copyText.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { SectionStrip } from '../shell/SectionStrip.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
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

function OfflineChecksNote() {
  const { t } = useLingui()
  return (
    <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
      {t`Some checks could not run without a connection, so more problems may show up later.`}
    </Typography>
  )
}

interface SectionAction {
  label: string
  icon?: ReactNode
  onClick: () => void
  disabled?: boolean
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
  const [addingAll, runAddAll] = usePending()
  const cosmeticConflicts = useOpenOverride(
    'cosmeticConflicts',
    useSettings((s) => gamePrefs(s).cosmeticConflicts),
  )
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
          icon: <Plus size={16} />,
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
    return {}
  }

  const problemsAt = useMods((s) => s.problemsAt)
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
      <ProblemActions />
      <SectionStrip
        label={t`Problem sections`}
        tabs={tabs}
        current={current}
        onChoose={(id) => choose(openId, id)}
        actions={
          shown?.action ? (
            <Button
              variant="outlined"
              color="inherit"
              disabled={shown.action.disabled}
              onClick={shown.action.onClick}
              startIcon={shown.action.icon}
              sx={{ height: space.control, borderColor: 'var(--mortar-hairline-20)' }}
            >
              {shown.action.label}
            </Button>
          ) : null
        }
      />
      <Box
        sx={{
          flex: empty ? 1 : '0 0 auto',
          px: space.gutter,
          py: space.gutter,
          display: 'flex',
          flexDirection: 'column',
          gap: space.gap,
        }}
      >
        <Box sx={{ mx: `calc(-1 * ${space.gutter})`, mb: -1 }}>
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
        {result.unknown ? <OfflineChecksNote /> : null}
        {empty ? null : <CheckTimings timings={result.timings ?? []} at={problemsAt} />}
      </Box>
    </>
  )
}

// ProblemActions ends the section strip's row.
function ProblemActions() {
  const { t } = useLingui()
  const result = useOpenProblems()
  const [mapOpen, setMapOpen] = useState(false)
  const hasAssetMap = useProfileLoader()?.assets
  const bisectBlocked = useBisectBlockText()
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
    <PageActions>
      <Tooltip
        title={bisectBlocked ?? t`Turns mods off in halves and relaunches until the crash stops`}
      >
        {/* A disabled button fires no pointer events, so the tooltip needs a live wrapper. */}
        <span>
          <Button
            variant="outlined"
            color="inherit"
            disabled={bisectBlocked !== null}
            onClick={findCrashCause}
            sx={{ height: space.control, borderColor: 'var(--mortar-hairline-20)' }}
          >
            {t`Find the mod that crashes the game…`}
          </Button>
        </span>
      </Tooltip>
      {hasAssetMap ? (
        <>
          <IconAction
            label={t`Asset map`}
            icon={<MapIcon size={16} />}
            onClick={() => setMapOpen(true)}
          />
          <AssetMapDialog open={mapOpen} onClose={() => setMapOpen(false)} />
        </>
      ) : null}
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
              copyText(text, t`Report copied`)
            })
            .catch(reportUnexpected)
        }}
      />
    </PageActions>
  )
}

function ProblemsTab() {
  const { t } = useLingui()
  const result = useOpenProblems()
  useLoadProblemsOnFocus()
  if (result === null) {
    return (
      <>
        <ProblemActions />
        <SectionStrip
          label={t`Problem sections`}
          tabs={[]}
          current=""
          onChoose={() => undefined}
          actions={null}
        />
        <SkeletonRows
          label={t`Checking the mods for problems…`}
          count={PROBLEM_SKELETON_ROWS}
          height={PROBLEM_SKELETON_HEIGHT}
          sx={{ px: space.gutter, py: space.gutter }}
        />
      </>
    )
  }
  return <ProblemsContent result={result} />
}

export { ProblemsTab }
