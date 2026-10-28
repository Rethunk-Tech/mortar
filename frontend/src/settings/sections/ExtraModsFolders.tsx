import { useLingui } from '@lingui/react/macro'
import { MenuItem, Select } from '@mui/material'
import {
  SetByKey,
  SetExtraModsProfile,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { installExtraFolder } from '../../downloads/autoInstall.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { useProfiles } from '../../profiles/store.ts'
import { saveSetting } from '../persist.ts'
import { useSettings } from '../store.ts'
import { FolderListRow } from './FolderListRow.tsx'
import { separator, split } from './pathList.ts'

const OPEN = 'open'

// Each folder installs its mods into the profile picked beside it, or the open one, now and whenever its files change.
function ExtraModsFolders() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const folders = split(useSettings((s) => s.games?.[game]?.extraModsFolder ?? ''))
  const targets = useSettings((s) => s.games?.[game]?.extraModsProfiles)
  const profiles = useProfiles((s) => (s.game?.id === game ? s.profiles : null))
  const save = (next: string[]) => SetByKey('extraModsFolder', next.join(separator), game)
  const target = (dir: string, profile: string) =>
    saveSetting(async () => {
      await SetExtraModsProfile(game, dir, profile === OPEN ? '' : profile)
      installExtraFolder(game, dir)
    })
  return (
    <FolderListRow
      label={t`Extra mods folders`}
      description={t`Folders of unpacked mods, or your own build's output. Their mods install into the profile beside each folder, and again whenever their files change.`}
      prompt={t`Extra mods folder`}
      folders={folders}
      save={save}
      onAdded={(dir) => installExtraFolder(game, dir)}
      beside={(dir) => (
        <Select
          size="small"
          value={targets?.[dir] || OPEN}
          inputProps={{ 'aria-label': t`Profile ${{ path: dir }} installs into` }}
          onChange={(e) => target(dir, e.target.value)}
        >
          <MenuItem value={OPEN}>{t`Open profile`}</MenuItem>
          {(profiles ?? []).map((p) => (
            <MenuItem key={p.id} value={p.id}>
              {p.name}
            </MenuItem>
          ))}
          {targets?.[dir] && profiles && !profiles.some((p) => p.id === targets[dir]) ? (
            <MenuItem value={targets[dir]}>{t`Deleted profile`}</MenuItem>
          ) : null}
        </Select>
      )}
    />
  )
}

export { ExtraModsFolders }
