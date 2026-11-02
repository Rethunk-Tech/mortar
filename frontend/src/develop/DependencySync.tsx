import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import {
  AddDependency,
  IgnoreDependency,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { listNames } from '../i18n/list.ts'
import { download } from '../queue/actions.ts'
import { FormSection } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { outOfStep, useDependencySync } from './dependencySync.ts'

/** Where the project's declared dependencies and its development profile's mods differ; every change is the player's click. */
export function DependencySync({
  game,
  dir,
  onChanged,
}: {
  game: string
  dir: string
  onChanged: () => void
}) {
  const { t } = useLingui()
  const { sync, reload } = useDependencySync(game, dir)
  if (!sync || outOfStep(sync) === 0) {
    return null
  }
  const done = () => {
    reload()
    onChanged()
  }
  const act = (run: () => Promise<unknown>, title: string) => () =>
    run().then(done).catch(reportError(title))
  const missing = sync.missing ?? []
  return (
    <FormSection
      title={t`Out of step`}
      description={t`The project and its development profile disagree. Nothing changes until you click.`}
    >
      {missing.length > 0 ? (
        <Row
          text={t`Not in the profile: ${{ names: listNames(missing.map((d) => d.package)) }}`}
          actions={
            <Button
              size="small"
              onClick={act(
                () =>
                  download(
                    missing.map((d) => ({
                      kind: 'install',
                      package: d.package,
                      version: d.version,
                    })),
                  ),
                t`Could not install the dependencies`,
              )}
            >
              {t`Install missing`}
            </Button>
          }
        />
      ) : null}
      {(sync.undeclared ?? []).map((p) => (
        <Row
          key={p.package}
          text={t`${{ name: p.package }} ${{ version: p.version }} is in the profile, not in the project`}
          actions={
            <>
              <Button
                size="small"
                onClick={act(
                  () => AddDependency(game, dir, p.package, p.version, false),
                  t`Could not add the dependency`,
                )}
              >
                {t`Add as dependency`}
              </Button>
              <Button
                size="small"
                color="inherit"
                onClick={act(() => IgnoreDependency(game, dir, p.package), t`Could not dismiss it`)}
              >
                {t`Dismiss`}
              </Button>
            </>
          }
        />
      ))}
      {(sync.differs ?? []).map((v) => (
        <Row
          key={v.package}
          text={t`${{ name: v.package }}: the project declares ${{ declared: v.declared }}, the profile has ${{ version: v.installed }}`}
          actions={
            <Button
              size="small"
              onClick={act(
                () => AddDependency(game, dir, v.package, v.installed, v.soft),
                t`Could not update the dependency`,
              )}
            >
              {t`Declare ${{ version: v.installed }}`}
            </Button>
          }
        />
      ))}
    </FormSection>
  )
}

function Row({ text, actions }: { text: string; actions: React.ReactNode }) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, flexWrap: 'wrap' }}>
      <Typography sx={{ flex: 1, minWidth: 0 }}>{text}</Typography>
      {actions}
    </Box>
  )
}
