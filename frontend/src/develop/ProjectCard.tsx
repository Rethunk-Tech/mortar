import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, Chip, CircularProgress, Menu, Typography } from '@mui/material'
import { relativeWhen, useNowValue } from '@rethunk/react-load-hooks'
import {
  Bug,
  Ellipsis,
  FolderOpen,
  FolderPlus,
  Hammer,
  PackagePlus,
  Pencil,
  ScrollText,
  SquareCode,
  Trash2,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  GitState,
  ProjectInfo,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Forget,
  OpenAssetFolder,
  OpenEditor,
  OpenProject,
  SetWatch,
  Trash as TrashProject,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { StartDebugging } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { formatNumber } from '../i18n/numberFormat.ts'
import { LetterTile } from '../mods/LetterTile.tsx'
import { useProfiles } from '../profiles/store.ts'
import { persist, saveSetting } from '../settings/persist.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { MenuAction, MenuCheck } from '../shell/MenuAction.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { MenuRule } from '../shell/TitleMenu.tsx'
import { usePopoverAnchor } from '../shell/usePopoverAnchor.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  buildProject,
  refreshInfo,
  runOf,
  useDevRuns,
  useProjectInfo,
  useTrustAsk,
} from './builds.ts'
import { outOfStep, useDependencySync } from './dependencySync.ts'
import { EditProjectDialog } from './EditProjectDialog.tsx'
import { useIsAsset } from './kinds.ts'
import { showModLog, useModCosts } from './modCosts.ts'
import { Outcome } from './Outcome.tsx'
import { ReleaseDialog } from './ReleaseDialog.tsx'
import { ReleaseItems } from './ReleaseItems.tsx'
import { UploadDialog } from './UploadDialog.tsx'

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
    return (
      <Chip
        size="small"
        sx={statusSx}
        color="error"
        label={plural(count, { one: '# error', other: '# errors' })}
      />
    )
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

// What the repository has not committed or pushed; nothing when both are clean.
function gitSummary(git: GitState | null | undefined): string {
  if (!git) {
    return ''
  }
  return [
    git.changed > 0 ? plural({ count: git.changed }, { one: '# changed', other: '# changed' }) : '',
    git.unpushed > 0
      ? plural({ count: git.unpushed }, { one: '# to push', other: '# to push' })
      : '',
  ]
    .filter(Boolean)
    .join(', ')
}

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

function MoreMenu({
  game,
  project,
  onRelease,
  onUpload,
  onEdit,
}: {
  game: string
  project: DevProject
  onRelease: () => void
  onUpload: (version: string) => void
  onEdit: () => void
}) {
  const { t } = useLingui()
  const { anchor, show, hide } = usePopoverAnchor()
  const [forgetting, setForgetting] = useState(false)
  const [trashFolder, setTrashFolder] = useState(false)
  const info = useProjectInfo((s) => s.byDir[project.dir])
  // A project made only of files has no build, plugin or log.
  const code = !useIsAsset(project.dir)
  // One start with the game's Mono debugger open; the toast says where to attach.
  const debug = () =>
    StartDebugging(game, project.profile)
      .then((address) =>
        useToasts.getState().push({
          kind: 'info',
          title: t`Starting for debugging`,
          body: t`Attach your editor's debugger to ${{ address }}.`,
        }),
      )
      .catch(reportError(t`Could not start the game`))
  const act = (fn: () => unknown) => () => {
    hide()
    fn()
  }
  return (
    <>
      <IconAction
        label={t`More actions`}
        icon={<Ellipsis size={18} />}
        menu={true}
        onClick={show}
      />
      <Menu anchorEl={anchor} open={anchor !== null} onClose={() => hide()}>
        <MenuAction
          icon={<Pencil size={16} aria-hidden={true} />}
          label={t`Edit details…`}
          disabled={!info}
          reason={t`Reading the project…`}
          onClick={act(onEdit)}
        />
        <MenuAction
          icon={<SquareCode size={16} aria-hidden={true} />}
          label={t`Open in editor`}
          onClick={act(() =>
            OpenEditor(game, project.dir).catch(reportError(t`Could not open the editor`)),
          )}
        />
        <MenuAction
          icon={<FolderOpen size={16} aria-hidden={true} />}
          label={t`Open folder`}
          onClick={act(() =>
            OpenProject(game, project.dir).catch(reportError(t`Could not open the folder`)),
          )}
        />
        {code ? <MenuRule /> : null}
        {code ? (
          <MenuAction
            icon={<Bug size={16} aria-hidden={true} />}
            label={t`Start for debugging`}
            onClick={act(debug)}
          />
        ) : null}
        {code ? (
          <MenuAction
            icon={<ScrollText size={16} aria-hidden={true} />}
            label={t`Show log`}
            disabled={!info}
            onClick={act(() => info && showModLog(info))}
          />
        ) : null}
        <MenuRule />
        <ReleaseItems
          act={act}
          released={info?.released}
          onRelease={onRelease}
          onUpload={onUpload}
        />
        {code ? <MenuRule /> : null}
        {code ? (
          <MenuCheck
            checked={Boolean(project.watch)}
            label={t`Build on save`}
            onClick={() => saveSetting(() => SetWatch(game, project.dir, !project.watch))}
          />
        ) : null}
        <MenuRule />
        <MenuAction
          icon={<Trash2 size={16} aria-hidden={true} />}
          label={t`Remove from list`}
          tone="error"
          onClick={act(() => setForgetting(true))}
        />
      </Menu>
      <ConfirmationDialog
        open={forgetting}
        tone={trashFolder ? 'danger' : 'warning'}
        title={t`Remove ${{ name: project.name }} from the list?`}
        body={
          trashFolder
            ? t`Its folder moves to the trash, where you can restore it, and the mod leaves the profile.`
            : t`Its folder stays where it is; Mortar stops building it and installing it into the profile.`
        }
        confirmLabel={trashFolder ? t`Remove and move to trash` : t`Remove from list`}
        onCancel={() => {
          setForgetting(false)
          setTrashFolder(false)
        }}
        onConfirm={() => {
          setForgetting(false)
          setTrashFolder(false)
          if (trashFolder) {
            TrashProject(game, project.dir).catch(
              reportError(t`Could not move the folder to the trash`),
            )
          } else {
            persist(() => Forget(game, project.dir), t`Could not remove the project`)
          }
        }}
        checkbox={{
          label: t`Also move its folder to the trash`,
          checked: trashFolder,
          onChange: setTrashFolder,
        }}
      >
        {trashFolder && (info?.git?.unpushed || info?.git?.changed) ? (
          <Alert variant="compact" severity="warning">
            {t`Work that is only in this folder goes with it: ${{ summary: gitSummary(info.git) }}.`}
          </Alert>
        ) : null}
      </ConfirmationDialog>
    </>
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
export function ProjectCard({
  game,
  project,
  list,
}: {
  game: string
  project: DevProject
  list: boolean
}) {
  const { t } = useLingui()
  const info = useInfo(game, project.dir)
  const [releasing, setReleasing] = useState(false)
  // The version a local release offers to upload; empty when none is open.
  const [uploading, setUploading] = useState('')
  const [editing, setEditing] = useState(false)
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
        <LetterTile
          mod={{ id: name, name, ...(info?.icon ? { picture: info.icon } : {}) }}
          size={40}
        />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, flexWrap: 'wrap' }}>
            <Typography sx={{ fontSize: 'large.fontSize', fontWeight: 600 }}>{name}</Typography>
            <Status project={project} />
            <OutOfStep game={game} dir={project.dir} />
          </Box>
          <Typography
            noWrap={true}
            title={more}
            sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}
          >
            {info ? details.join(' · ') : project.dir}
          </Typography>
          <Costs game={game} profile={project.profile} dir={project.dir} info={info} />
        </Box>
        <BuildButton game={game} project={project} compact={!list} />
        <MoreMenu
          game={game}
          project={project}
          onRelease={() =>
            project.untrusted
              ? useTrustAsk.setState({ ask: { game, dir: project.dir, pack: false } })
              : setReleasing(true)
          }
          onUpload={setUploading}
          onEdit={() => setEditing(true)}
        />
      </Box>
      <Outcome game={game} project={project} />
      {editing && info ? (
        <EditProjectDialog
          game={game}
          dir={project.dir}
          info={info}
          onClose={() => setEditing(false)}
        />
      ) : null}
      {releasing ? (
        <ReleaseDialog
          game={game}
          dir={project.dir}
          name={name}
          onClose={() => setReleasing(false)}
          onLocal={(released) => {
            setReleasing(false)
            setUploading(released)
          }}
        />
      ) : null}
      {uploading ? (
        <UploadDialog
          game={game}
          dir={project.dir}
          name={name}
          version={uploading}
          onClose={() => setUploading('')}
        />
      ) : null}
    </Box>
  )
}
