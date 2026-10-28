import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { space } from '../../theme/density.ts'
import { saveSetting } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'

/** A setting that holds a list of folders: each with a Remove button (and `beside` before it), then Add folder…. */
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
  return (
    <SettingRow label={label} description={description}>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-start' }}>
        {folders.map((dir) => (
          <Box key={dir} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Typography noWrap={true} title={dir} sx={{ maxWidth: 360 }}>
              {dir}
            </Typography>
            {beside?.(dir)}
            <Button
              size="small"
              aria-label={t`Remove ${{ name: dir }}`}
              onClick={() => saveSetting(() => save(folders.filter((d) => d !== dir)))}
            >
              {t`Remove`}
            </Button>
          </Box>
        ))}
        <Button variant="outlined" onClick={add}>
          {t`Add folder…`}
        </Button>
      </Box>
    </SettingRow>
  )
}
