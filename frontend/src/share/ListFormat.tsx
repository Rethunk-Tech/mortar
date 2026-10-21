import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ToggleButton, ToggleButtonGroup, Typography } from '@mui/material'
import { Check, Copy, FileText, MessageSquare, Package, Save, Type } from 'lucide-react'
import { useState } from 'react'
import { SaveFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { safeFileName } from '../console/save.ts'
import { listNames } from '../i18n/list.ts'
import { useProfiles } from '../profiles/store.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { copyText } from './copyText.ts'
import { formatModList, listItems, type ModListFormat, skippedForDependencies } from './modList.ts'
import { useShareDialog } from './store.ts'

const text = {
  m: 0,
  p: space.pad,
  fontFamily: MONO,
  fontSize: 13,
  lineHeight: 1.6,
  whiteSpace: 'pre-wrap',
  overflowWrap: 'anywhere',
  userSelect: 'text',
} as const

function useFormatOptions(): { id: ModListFormat; label: string; icon: typeof FileText }[] {
  const { t } = useLingui()
  return [
    { id: 'markdown', label: t`Markdown`, icon: FileText },
    { id: 'plain', label: t`Plain text`, icon: Type },
    { id: 'discord', label: sourceLabel('discord'), icon: MessageSquare },
    { id: 'thunderstore', label: sourceLabel('thunderstore'), icon: Package },
  ]
}

function ListHint({ skipped, total }: { skipped: string[]; total: number }) {
  const { t } = useLingui()
  const names = listNames(skipped)
  let hint = t`Paste it into a chat or a forum post, or send the file.`
  if (skipped.length > 0) {
    hint = t`Left out (not on Thunderstore, or switched off): ${names}`
  } else if (total > 1) {
    hint = t`Discord takes 2,000 characters a message: paste each one in turn.`
  }
  return <Typography sx={{ flex: 1, fontSize: 13, color: 'text.secondary' }}>{hint}</Typography>
}

// The mod list as text: the format, the text itself as it will be pasted, and Copy and Save. Discord's list is
// shown as the messages it is split into, copied one after another.
export function ListFormat() {
  const { t, i18n } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const keys = useShareDialog((s) => s.keys)
  const format = useShareDialog((s) => s.listFormat)
  const setFormat = useShareDialog((s) => s.setListFormat)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === profileId))
  const [copied, setCopied] = useState(0)
  const items = listItems(profile, keys)
  const labels = { enabled: t`Enabled`, disabled: t`Disabled` }
  const parts = formatModList(format, items, labels)
  const total = parts.length
  const skipped = format === 'thunderstore' ? skippedForDependencies(items) : []
  const next = copied >= total ? 0 : copied
  const off = items.filter((item) => !item.enabled).length
  const options = useFormatOptions()
  const count = items.length
  const summary = [
    plural(count, { one: '# mod', other: '# mods' }),
    off > 0 ? t`${off} switched off` : '',
    total > 1 ? t`${total} messages` : '',
  ]
    .filter(Boolean)
    .join(' · ')
  const copyOne = (n: number, body: string) =>
    copyText(body, total > 1 ? t`Message ${n} of ${total} copied` : t`Mod list copied`)
  const copyNext = () => {
    copyOne(next + 1, parts[next]?.text ?? '')
    setCopied(next + 1)
  }
  // A file has no message limit, so Discord's list is saved whole, as the Markdown it is.
  const save = () => {
    const whole = formatModList(format === 'discord' ? 'markdown' : format, items, labels)
    SaveFile(
      t`Save mod list`,
      `${safeFileName(profile?.name ?? '', 'profile')} mods.txt`,
      whole[0]?.text ?? '',
    )
      .then((path) => {
        if (path) {
          useToasts.getState().push({ kind: 'success', title: t`Mod list saved`, body: path })
        }
      })
      .catch((error: unknown) => toastError(t`Could not save the mod list`, error))
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, gap: space.pad }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.pad }}>
        <ToggleButtonGroup
          exclusive={true}
          size="small"
          value={format}
          aria-label={t`Mod list format`}
          onChange={(_, chosen: ModListFormat | null) => {
            if (chosen) {
              setFormat(chosen)
              setCopied(0)
            }
          }}
        >
          {options.map((o) => (
            <ToggleButton key={o.id} value={o.id} sx={{ px: space.pad, gap: '8px' }}>
              <o.icon size={15} />
              {o.label}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
        <Box sx={{ flex: 1 }} />
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{summary}</Typography>
      </Box>
      <Box
        role="region"
        aria-label={t`The text that is copied`}
        tabIndex={0}
        sx={{
          flex: '1 1 0',
          minHeight: 160,
          overflowY: 'auto',
          bgcolor: 'var(--mortar-overlay-30)',
          border: '1px solid var(--mortar-hairline-muted)',
          borderRadius: '8px',
        }}
      >
        {parts.map((part) => (
          <Box key={part.id}>
            {total > 1 ? (
              <Box
                sx={{
                  position: 'sticky',
                  top: 0,
                  display: 'flex',
                  alignItems: 'center',
                  gap: space.gap,
                  pl: space.pad,
                  pr: '6px',
                  height: 36,
                  bgcolor: 'var(--mortar-panel-solid)',
                  borderBottom: '1px solid var(--mortar-hairline-muted)',
                }}
              >
                {part.n <= copied ? <Check size={14} aria-label={t`Copied`} /> : null}
                <Typography sx={{ flex: 1, fontSize: 12, fontWeight: 600 }}>
                  {t`Message ${part.n} of ${total}`}
                  <Typography
                    component="span"
                    sx={{ ml: 1, fontSize: 12, color: 'text.secondary' }}
                  >
                    {t`${part.text.length.toLocaleString(i18n.locale)} characters`}
                  </Typography>
                </Typography>
                <TipIconButton
                  label={t`Copy message ${part.n}`}
                  onClick={() => copyOne(part.n, part.text)}
                >
                  <Copy size={14} />
                </TipIconButton>
              </Box>
            ) : null}
            <Box component="pre" sx={text}>
              {part.text}
            </Box>
          </Box>
        ))}
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <ListHint skipped={skipped} total={total} />
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<Save size={16} />}
          onClick={save}
          sx={{ height: 40, px: '14px', fontSize: 14 }}
        >
          {t`Save to a file…`}
        </Button>
        <Button
          variant="contained"
          startIcon={<Copy size={16} />}
          onClick={copyNext}
          sx={{ height: 40, px: '18px', fontSize: 14, whiteSpace: 'nowrap' }}
        >
          {total > 1 ? t`Copy message ${next + 1} of ${total}` : t`Copy mod list`}
        </Button>
      </Box>
    </Box>
  )
}
