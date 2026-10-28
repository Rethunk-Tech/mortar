import type { useLingui } from '@lingui/react/macro'
import { Tag, Upload } from 'lucide-react'
import { MenuAction } from '../shell/MenuAction.tsx'

/** The Release… item, and Upload… for the latest released version, which a release that stayed local or failed to
 * upload leaves unpublished. */
export function releaseItems(
  act: (fn: () => unknown) => () => void,
  released: string | undefined,
  onRelease: () => void,
  onUpload: (version: string) => void,
  t: ReturnType<typeof useLingui>['t'],
) {
  return [
    <MenuAction
      key="release"
      icon={<Tag size={16} aria-hidden={true} />}
      label={t`Release…`}
      onClick={act(onRelease)}
    />,
    released ? (
      <MenuAction
        key="upload"
        icon={<Upload size={16} aria-hidden={true} />}
        label={t`Upload v${{ version: released }}…`}
        onClick={act(() => onUpload(released))}
      />
    ) : null,
  ]
}
