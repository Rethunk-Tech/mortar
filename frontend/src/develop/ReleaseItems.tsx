import { useLingui } from '@lingui/react/macro'
import { Tag, Upload } from 'lucide-react'
import { MenuAction } from '../shell/MenuAction.tsx'

/** The Release… item, and Upload… for the latest released version, which a release that stayed local or failed to
 * upload leaves unpublished. */
export function ReleaseItems({
  act,
  released,
  onRelease,
  onUpload,
}: {
  act: (fn: () => unknown) => () => void
  released: string | undefined
  onRelease: () => void
  onUpload: (version: string) => void
}) {
  const { t } = useLingui()
  return (
    <>
      <MenuAction
        icon={<Tag size={16} aria-hidden={true} />}
        label={t`Release…`}
        onClick={act(onRelease)}
      />
      {released ? (
        <MenuAction
          icon={<Upload size={16} aria-hidden={true} />}
          label={t`Upload v${{ version: released }}…`}
          onClick={act(() => onUpload(released))}
        />
      ) : null}
    </>
  )
}
