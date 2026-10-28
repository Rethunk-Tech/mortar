import { useLingui } from '@lingui/react/macro'
import { Link2, Link2Off } from 'lucide-react'
import { type Tile, TileGrid } from '../share/TileGrid.tsx'
import { AppDialog } from '../shell/dialog/AppDialog.tsx'
import { addDependency, useDependencyAsk } from './dependencyAsk.ts'

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
    addDependency(ask, kind === 'soft')
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
