import { useLingui } from '@lingui/react/macro'
import { Link2, Link2Off } from 'lucide-react'
import { AddDependency } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { type Tile, TileGrid } from '../share/TileGrid.tsx'
import { AppDialog } from '../shell/dialog/AppDialog.tsx'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useDependencyAsk } from './dependencyAsk.ts'

const ICON_PX = 32

/** Asks whether a project needs a package (it joins the manifest) or only uses it when present. */
export function DependencyKindDialog() {
  const { t } = useLingui()
  const ask = useDependencyAsk((s) => s.ask)
  const close = () => useDependencyAsk.setState({ ask: null })
  if (!ask) {
    return null
  }
  const tiles: Tile<'hard' | 'soft'>[] = [
    {
      id: 'hard',
      icon: <Link2 size={ICON_PX} />,
      name: t`Required`,
      hint: t`Listed in the manifest; mod managers install it with ${{ name: ask.project }}`,
    },
    {
      id: 'soft',
      icon: <Link2Off size={ICON_PX} />,
      name: t`Optional`,
      hint: t`${{ name: ask.project }} runs without it and uses it when it is there`,
    },
  ]
  const choose = (kind: 'hard' | 'soft') => {
    close()
    ask.install?.()
    AddDependency(ask.game, ask.dir, ask.pkg, ask.version, kind === 'soft')
      .then(() =>
        useToasts.getState().push({
          kind: 'success',
          title:
            kind === 'soft'
              ? t`${{ mod: ask.name }} is an optional dependency of ${{ name: ask.project }}`
              : t`${{ mod: ask.name }} is a dependency of ${{ name: ask.project }}`,
        }),
      )
      .catch(reportError(t`Could not add the dependency`))
  }
  return (
    <AppDialog
      open={true}
      onClose={close}
      title={t`How does ${{ name: ask.project }} use ${{ mod: ask.name }}?`}
      subtitle={t`Either way its files are referenced for building, not copied into your package.`}
      size="md"
      flush={true}
    >
      <TileGrid label={t`Dependency kind`} tiles={tiles} lastUsed={null} onChoose={choose} />
    </AppDialog>
  )
}
