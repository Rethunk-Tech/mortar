import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Chip, CircularProgress, Typography } from '@mui/material'
import { relativeWhen, useNowValue } from '@rethunk/react-load-hooks'
import { FolderPlus, Hammer, PackagePlus } from 'lucide-react'
import { type ReactNode, useEffect } from 'react'
import type { ProjectInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Forget,
  OpenAssetFolder,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { errorsLabel } from '../i18n/counts.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { formatNumber } from '../i18n/numberFormat.ts'
import { LetterTile } from '../mods/LetterTile.tsx'
import { useProfiles } from '../profiles/store.ts'
import { persist } from '../settings/persist.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { buildProject, refreshInfo, runOf, useDevRuns, useProjectInfo } from './builds.ts'
import { outOfStep, useDependencySync } from './dependencySync.ts'
import { useIsAsset } from './kinds.ts'
import { useModCosts } from './modCosts.ts'
import { Outcome } from './Outcome.tsx'
import { ProjectMenu } from './ProjectMenu.tsx'
import { gitSummary } from './projectSummary.ts'

// Whether the project's folder is in its profile: a mod removed there stays out until asked for again.
function useInProfile(project: DevProject) {
  return useProfiles(
    (s) =>
      s.profiles
        .find((p) => p.id === project.profile)
        ?.entries?.some((e) => e.source.folder === project.output) ?? true,
  )
}

function Status({ project }: { project: DevProject }) {
  const dir = project.dir
  const asset = useIsAsset(dir)
  const inProfile = useInProfile(project)
  const { t, i18n } = useLingui()
  const run = useDevRuns((s) => runOf(s.byDir, dir))
  // The files remember a build from before Mortar started; this session's result adds its errors.
  const disk = useProjectInfo((s) => s.byDir[dir])
  const builtAt = Math.max(run.builtAt, disk?.builtAt ?? 0)
  const when = useNowValue((now) => relativeWhen(builtAt, { now, locale: i18n.locale }))
  if (run.running) {
    return (
      <Chip
        size="small"
        sx={statusSx}
        icon={<CircularProgress size={10} />}
        label={run.running === 'build' ? t`Building` : t`Packing`}
      />
    )
  }
  if (run.build && !run.build.ok) {
    const count = (run.build.problems ?? []).filter((p) => p.severity === 'error').length
    return <Chip size="small" sx={statusSx} color="error" label={errorsLabel(count)} />
  }
  if (asset) {
    return project.untrusted || !inProfile ? (
      <Chip size="small" sx={statusSx} variant="outlined" label={t`Not installed`} />
    ) : (
      <Chip size="small" sx={statusSx} color="success" variant="outlined" label={t`Installed`} />
    )
  }
  if (!builtAt) {
    return <Chip size="small" sx={statusSx} variant="outlined" label={t`Not built yet`} />
  }
  if (disk?.stale) {
    return (
      <Chip
        size="small"
        sx={statusSx}
        color="warning"
        variant="outlined"
        label={t`Changed since the last build`}
      />
    )
  }
  return (
    <Chip
      size="small"
      sx={statusSx}
      color="success"
      variant="outlined"
      label={t`Built ${{ when }}`}
    />
  )
}

// Counts where the project's dependencies and its profile's mods disagree; the Edit dialog's Dependencies tab lists them.
function OutOfStep({ game, dir }: { game: string; dir: string }) {
  const { t } = useLingui()
  const count = outOfStep(useDependencySync(game, dir).sync)
  return count > 0 ? (
    <Chip
      size="small"
      sx={statusSx}
      color="warning"
      variant="outlined"
      label={plural(count, {
        one: '# dependency out of step',
        other: '# dependencies out of step',
      })}
      title={t`Open Edit, Dependencies to review`}
    />
  ) : null
}

const statusSx = {
  height: 20,
  fontSize: 'tiny.fontSize',
  '& .MuiChip-label': { px: 0.75 },
} as const

// The project's details from its own files, read when the card shows and kept current by refreshInfo.
function useInfo(game: string, dir: string) {
  const info = useProjectInfo((s) => s.byDir[dir])
  useEffect(() => {
    refreshInfo(game, dir).catch(() => undefined)
  }, [game, dir])
  return info
}

// The mod's own cost in the development profile's latest measured run.
function Costs({
  game,
  profile,
  dir,
  info,
}: {
  game: string
  profile: string
  dir: string
  info: ProjectInfo | undefined
}) {
  const { t, i18n } = useLingui()
  const run = useDevRuns((s) => runOf(s.byDir, dir))
  // As the card's Built label counts it: a build that changed nothing still leaves the output's time as it was.
  const builtAt = Math.max(run.builtAt, info?.builtAt ?? 0)
  const { startupMs, startupAt, frameMs, frameAt } = useModCosts(game, profile, info, builtAt)
  const start = startupMs === undefined ? '' : formatNumber(Math.round(startupMs))
  const frame =
    frameMs === undefined ? '' : formatNumber(frameMs, i18n.locale, { maximumFractionDigits: 2 })
  const line = [
    start ? t`Measured ${{ when: formatWhen(startupAt ?? '') }}: ${{ start }} ms to start` : '',
    frame ? t`Measured ${{ when: formatWhen(frameAt ?? '') }}: ${{ frame }} ms a frame` : '',
  ]
    .filter(Boolean)
    .join(' · ')
  return line ? (
    <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>{line}</Typography>
  ) : null
}

// A project whose folder or project file is gone: it can only leave the list.
function MissingCard({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  return (
    <Alert
      variant="compact"
      severity="warning"
      action={
        <Button
          size="small"
          onClick={() => persist(() => Forget(game, project.dir), t`Could not remove the project`)}
        >
          {t`Remove from list`}
        </Button>
      }
    >
      {t`${{ name: project.name }} is no longer in ${{ path: project.dir }}.`}
    </Alert>
  )
}

function BuildButton({
  game,
  project,
  compact,
}: {
  game: string
  project: DevProject
  compact: boolean
}) {
  const { t } = useLingui()
  const running = useDevRuns((s) => runOf(s.byDir, project.dir).running)
  const build = () => buildProject(game, project.dir).catch(() => undefined)
  const asset = useIsAsset(project.dir)
  const inProfile = useInProfile(project)
  if (asset && (project.untrusted || !inProfile)) {
    // Its files install through a build; one added from a folder has its refusal ask for trust first.
    return compact ? (
      <TipIconButton label={t`Install`} onClick={build}>
        <PackagePlus size={18} />
      </TipIconButton>
    ) : (
      <Button variant="outlined" startIcon={<PackagePlus size={16} />} onClick={build}>
        {t`Install`}
      </Button>
    )
  }
  if (asset) {
    // A project made only of files installs as it is; its button opens the folder the loader reads.
    const open = () =>
      OpenAssetFolder(game, project.dir).catch(reportError(t`Could not open the folder`))
    return compact ? (
      <TipIconButton label={t`Add files`} onClick={open}>
        <FolderPlus size={18} />
      </TipIconButton>
    ) : (
      <Button variant="outlined" startIcon={<FolderPlus size={16} />} onClick={open}>
        {t`Add files`}
      </Button>
    )
  }
  if (compact) {
    return (
      <TipIconButton label={t`Build`} disabled={running !== null} onClick={build}>
        <Hammer size={18} />
      </TipIconButton>
    )
  }
  return (
    <Button
      variant="outlined"
      startIcon={<Hammer size={16} />}
      disabled={running !== null}
      onClick={build}
    >
      {t`Build`}
    </Button>
  )
}

// One project: its name and state beside its actions; a grid card's narrower Build is an icon.
function StatusChips({ game, project }: { game: string; project: DevProject }) {
  return (
    <>
      <Status project={project} />
      <OutOfStep game={game} dir={project.dir} />
    </>
  )
}

const rowSx = {
  display: 'flex',
  alignItems: 'center',
  gap: space.gap,
  '& > .MuiTypography-root': { flexShrink: 1 },
  '& > :not(.MuiTypography-root)': { flexShrink: 0 },
} as const

export function ProjectCard({
  game,
  project,
  list,
  handle,
}: {
  game: string
  project: DevProject
  list: boolean
  handle?: ReactNode
}) {
  const { t } = useLingui()
  const info = useInfo(game, project.dir)
  if (info?.missing) {
    return <MissingCard game={game} project={project} />
  }
  const name = info?.name ?? project.name
  const version = info?.released ? `v${info.released}` : t`Unreleased`
  const details = [info ? version : '', gitSummary(info?.git)].filter(Boolean)
  // The card's line stays short; the id, branch and folder are a hover away.
  const more = [info?.guid, info?.git?.branch, project.dir].filter(Boolean).join('\n')
  return (
    <Box
      sx={{
        p: space.pad,
        bgcolor: 'var(--mortar-overlay-45)',
        borderRadius: '6px',
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        {handle}
        <LetterTile
          mod={{ id: name, name, ...(info?.icon ? { picture: info.icon } : {}) }}
          size={40}
        />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          {/* Each card is two lines in the grid, so a row's cards match: the status chips sit beside the version there,
              leaving the name the whole first line. */}
          <Box sx={rowSx}>
            <Typography
              noWrap={true}
              title={name}
              sx={{ fontSize: 'large.fontSize', fontWeight: 600, minWidth: 0 }}
            >
              {name}
            </Typography>
            {list ? <StatusChips game={game} project={project} /> : null}
          </Box>
          <Box sx={rowSx}>
            {list ? null : <StatusChips game={game} project={project} />}
            <Typography
              noWrap={true}
              title={more}
              sx={{ fontSize: 'small.fontSize', color: 'text.secondary', minWidth: 0 }}
            >
              {info ? details.join(' · ') : project.dir}
            </Typography>
          </Box>
          {/* Only some projects have a measured run, so in the grid the line would make cards of one row differ. */}
          {list ? (
            <Costs game={game} profile={project.profile} dir={project.dir} info={info} />
          ) : null}
        </Box>
        <BuildButton game={game} project={project} compact={!list} />
        <ProjectMenu game={game} project={project} info={info} name={name} />
      </Box>
      <Outcome game={game} project={project} />
    </Box>
  )
}
