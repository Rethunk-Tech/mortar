import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, FormControlLabel, Link, Switch, Typography } from '@mui/material'
import { useState } from 'react'
import type { Problem } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Forget,
  OpenAt,
  OpenProject,
  SetWatch,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { StartInstances } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { OpenWeb } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { PageActions } from '../game/PageActions.tsx'
import { useProfiles } from '../profiles/store.ts'
import { persist } from '../settings/persist.ts'
import { useSettings } from '../settings/store.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { buildProject, packProject, runOf, useDevRuns } from './builds.ts'
import { NewModDialog } from './NewModDialog.tsx'
import { ReleaseDialog } from './ReleaseDialog.tsx'

const THUNDERSTORE_UPLOAD = 'https://thunderstore.io/package/create/'
const TEST_COPIES = 2

// A problem's file below the project, as the editor would show it.
const relative = (dir: string, file: string) =>
  file.startsWith(`${dir}/`) || file.startsWith(`${dir}\\`) ? file.slice(dir.length + 1) : file

function Problems({
  game,
  project,
  problems,
}: {
  game: string
  project: DevProject
  problems: Problem[]
}) {
  const { t } = useLingui()
  const failed = t`Could not open the file`
  return (
    <Box component="ul" sx={{ m: 0, pl: 2.5, fontFamily: 'monospace', fontSize: 'small.fontSize' }}>
      {problems.map((p) => (
        <li key={`${p.file}:${p.line}:${p.column}:${p.code}`}>
          <Link
            component="button"
            color={p.severity === 'error' ? 'error' : 'warning'}
            onClick={() =>
              OpenAt(game, project.dir, p.file, p.line, p.column).catch(reportError(failed))
            }
            sx={{ textAlign: 'left' }}
          >
            {`${relative(project.dir, p.file)}(${p.line},${p.column}) ${p.code}: ${p.message}`}
          </Link>
        </li>
      ))}
    </Box>
  )
}

function Outcome({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  const run = useDevRuns((s) => runOf(s.byDir, project.dir))
  const build = run.build
  const pack = run.pack
  return (
    <>
      {build ? (
        <Alert variant="compact" severity={build.ok ? 'success' : 'error'}>
          {build.ok ? t`Built; it installs into the development profile.` : t`The build failed.`}
        </Alert>
      ) : null}
      {build && (build.problems ?? []).length > 0 ? (
        <Problems game={game} project={project} problems={build.problems ?? []} />
      ) : null}
      {pack?.zip && (pack.issues ?? []).length === 0 ? (
        <Alert
          variant="compact"
          severity="success"
          action={
            <Button
              size="small"
              onClick={() =>
                OpenWeb(THUNDERSTORE_UPLOAD).catch(reportError(t`Could not open the page`))
              }
            >
              {t`Upload…`}
            </Button>
          }
        >
          {t`Ready to upload: ${{ path: pack.zip }}`}
        </Alert>
      ) : null}
      {pack && (pack.issues ?? []).length > 0 ? (
        <Alert variant="compact" severity="warning">
          {(pack.issues ?? []).join(' ')}
        </Alert>
      ) : null}
    </>
  )
}

function ProjectCard({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  const running = useDevRuns((s) => runOf(s.byDir, project.dir).running)
  const profileName = useProfiles(
    (s) => s.profiles.find((p) => p.id === project.profile)?.name ?? '',
  )
  const [releasing, setReleasing] = useState(false)
  const failed = t`Could not save that setting`
  const test = () => {
    useProfiles.getState().open(project.profile)
    StartInstances(game, project.profile, TEST_COPIES).catch(
      reportError(t`Could not start the game`),
    )
  }
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
      <Box sx={{ display: 'flex', alignItems: 'baseline', gap: space.gap, minWidth: 0 }}>
        <Typography sx={{ fontSize: 'large.fontSize', fontWeight: 600 }}>{project.name}</Typography>
        <Typography
          noWrap={true}
          sx={{ fontSize: 'small.fontSize', color: 'text.secondary', minWidth: 0 }}
          title={project.dir}
        >
          {profileName ? t`Installs into ${{ profile: profileName }}` : project.dir}
        </Typography>
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: space.gap, alignItems: 'center' }}>
        <Button
          variant="outlined"
          disabled={running !== null}
          onClick={() => buildProject(game, project.dir).catch(() => undefined)}
        >
          {running === 'build' ? t`Building` : t`Build`}
        </Button>
        <FormControlLabel
          control={
            <Switch
              checked={Boolean(project.watch)}
              onChange={(_, on) => persist(() => SetWatch(game, project.dir, on), failed)}
            />
          }
          label={t`Build on save`}
        />
        <Button
          disabled={running !== null}
          onClick={() => packProject(game, project.dir).catch(() => undefined)}
        >
          {running === 'pack' ? t`Packing` : t`Pack`}
        </Button>
        <Button onClick={() => setReleasing(true)}>{t`Release…`}</Button>
        <Button onClick={test}>{t`Test with ${{ count: TEST_COPIES }} copies`}</Button>
        <Button
          onClick={() =>
            OpenProject(game, project.dir).catch(reportError(t`Could not open the folder`))
          }
        >
          {t`Open folder`}
        </Button>
        <Button
          onClick={() => persist(() => Forget(game, project.dir), failed)}
        >{t`Remove from list`}</Button>
      </Box>
      <Outcome game={game} project={project} />
      {releasing ? (
        <ReleaseDialog
          game={game}
          dir={project.dir}
          name={project.name}
          onClose={() => setReleasing(false)}
        />
      ) : null}
    </Box>
  )
}

/** The player's own mods for this game: start one from the template, then build, test, pack and release it. */
export function DevelopTab({ game }: { game: string }) {
  const { t } = useLingui()
  const projects = useSettings((s) => s.games?.[game]?.devProjects) ?? []
  const [creating, setCreating] = useState(false)
  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        overflowY: 'auto',
        p: space.gutter,
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
      }}
    >
      <PageActions>
        <Button variant="contained" onClick={() => setCreating(true)}>
          {t`New mod…`}
        </Button>
      </PageActions>
      {projects.length === 0 ? (
        <Typography sx={{ color: 'text.secondary' }}>
          {t`Start a mod from the template: Mortar sets up the project and installs each build into a development profile.`}
        </Typography>
      ) : null}
      {projects.map((p) => (
        <ProjectCard key={p.dir} game={game} project={p} />
      ))}
      <NewModDialog open={creating} game={game} onClose={() => setCreating(false)} />
    </Box>
  )
}
