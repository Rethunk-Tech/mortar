import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { TipIconButton } from '../../shell/TipIconButton.tsx'
import { space } from '../../theme/density.ts'
import { saveSetting } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { distinctTails } from './pathList.ts'

/**
 * A setting that holds a list of folders, under its label: each shows the ending that tells it apart (the whole path on
 * hover), then `beside` and a remove button, then Add folder….
 */
export function FolderListRow({
  label,
  description,
  prompt,
  folders,
  save,
  onAdded,
  beside,
}: {
  label: string
  description: string
  prompt: string
  folders: string[]
  save: (next: string[]) => Promise<void>
  onAdded?: (dir: string) => void
  beside?: (dir: string) => ReactNode
}) {
  const { t } = useLingui()
  const add = () =>
    saveSetting(async () => {
      const dir = await PickFolder(prompt)
      if (dir && !folders.includes(dir)) {
        await save([...folders, dir])
        onAdded?.(dir)
      }
    })
  const tails = distinctTails(folders)
  return (
    <SettingRow label={label} description={description} block={true}>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {folders.map((dir, i) => (
          <Box
            key={dir}
            sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minWidth: 0 }}
          >
            <Typography noWrap={true} title={dir} sx={{ flex: 1, minWidth: 0 }}>
              {tails[i]}
            </Typography>
            {beside?.(dir)}
            <TipIconButton
              label={t`Remove ${{ name: dir }}`}
              tip={t`Remove`}
              onClick={() => saveSetting(() => save(folders.filter((d) => d !== dir)))}
            >
              <X size={16} />
            </TipIconButton>
          </Box>
        ))}
        <Box>
          <Button variant="outlined" onClick={add}>
            {t`Add folder…`}
          </Button>
        </Box>
      </Box>
    </SettingRow>
  )
}
