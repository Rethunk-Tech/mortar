import { useLingui } from '@lingui/react/macro'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../store.ts'
import { FolderListRow } from './FolderListRow.tsx'
import { separator, split } from './pathList.ts'

function WatchFolders() {
  const { t } = useLingui()
  return (
    <FolderListRow
      label={t`Also watch these folders`}
      description={t`New archives in these folders are offered for install. Subfolders are not watched. For a mod's build output, use the game's Extra mods folders.`}
      prompt={t`Folder to watch`}
      folders={split(useSettings((s) => s.watchFolders ?? ''))}
      save={(next) => SetByKey('watchFolders', next.join(separator), '')}
    />
  )
}

export { WatchFolders }
