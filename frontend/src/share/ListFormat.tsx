import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ToggleButton, ToggleButtonGroup } from '@mui/material'
import { Check, Copy, FileText, MessageSquare, Package, Save, Type } from 'lucide-react'
import { useState } from 'react'
import { SaveFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { safeFileName } from '../console/save.ts'
import { listNames } from '../i18n/list.ts'
import { profileById, useProfiles } from '../profiles/store.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { copyText } from './copyText.ts'
import { formatModList, listItems, type ModListFormat, skippedForDependencies } from './modList.ts'
import { useShareDialog } from './store.ts'

// The parts are plain elements styled from the root: picking a format redraws the whole pane, and each Box or
// Typography would be three components more each time.
const paneSx = {
  display: 'flex',
  flexDirection: 'column',
  flex: 1,
  minHeight: 0,
  gap: space.pad,
  '& .list-bar, & .list-actions': { display: 'flex', alignItems: 'center', gap: space.pad },
  '& .list-actions': { gap: space.gap },
  '& .list-note': { m: 0, fontSize: 'base.fontSize', color: 'text.secondary' },
  '& .list-actions .list-note': { flex: 1 },
  '& .list-bar .list-note': { ml: 'auto' },
  '& .list-text': {
    flex: '1 1 0',
    minHeight: 160,
    overflowY: 'auto',
    bgcolor: 'var(--mortar-overlay-30)',
    border: '1px solid var(--mortar-hairline-muted)',
    borderRadius: '8px',
  },
  '& .list-message': {
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
  },
  '& .list-message-title': { flex: 1, m: 0, fontSize: 'small.fontSize', fontWeight: 600 },
  '& .list-message-size': { ml: 1, color: 'text.secondary', fontWeight: 400 },
  '& pre': {
    m: 0,
    p: space.pad,
    fontFamily: MONO,
    fontSize: 'base.fontSize',
    lineHeight: 1.6,
    whiteSpace: 'pre-wrap',
    overflowWrap: 'anywhere',
    userSelect: 'text',
  },
} as const

// Thunderstore's dependency list is offered only for a game whose mods come from Thunderstore.
function useFormatOptions(
  thunderstore: boolean,
): { id: ModListFormat; label: string; icon: typeof FileText }[] {
  const { t } = useLingui()
  return [
    { id: 'markdown' as const, label: t`Markdown`, icon: FileText },
    { id: 'plain' as const, label: t`Plain text`, icon: Type },
    { id: 'discord' as const, label: sourceLabel('discord'), icon: MessageSquare },
    ...(thunderstore
      ? [{ id: 'thunderstore' as const, label: sourceLabel('thunderstore'), icon: Package }]
      : []),
  ]
}

function ListHint({ skipped, total }: { skipped: string[]; total: number }) {
  const { t } = useLingui()
  const names = listNames(skipped)
  let hint = t`Paste it into a chat or a forum post, or send the file.`
  if (skipped.length > 0) {
    hint = t`Left out (not on ${{ source: sourceLabel('thunderstore') }}, or disabled): ${names}`
  } else if (total > 1) {
    hint = t`${{ app: sourceLabel('discord') }} takes 2,000 characters a message: paste each one in turn.`
  }
  return <p className="list-note">{hint}</p>
}

// The mod list as text: the format, the text itself as it will be pasted, and Copy and Save. Discord's list is
// shown as the messages it is split into, copied one after another.
export function ListFormat() {
  const { t, i18n } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const keys = useShareDialog((s) => s.keys)
  const thunderstore = useProfiles((s) => s.game?.sources?.includes('thunderstore') === true)
  const chosenFormat = useShareDialog((s) => s.listFormat)
  const format = chosenFormat === 'thunderstore' && !thunderstore ? 'markdown' : chosenFormat
  const setFormat = useShareDialog((s) => s.setListFormat)
  const profile = useProfiles(profileById(profileId))
  const [copied, setCopied] = useState(0)
  const items = listItems(profile, keys)
  const labels = { enabled: t`Enabled`, disabled: t`Disabled` }
  const parts = formatModList(format, items, labels)
  const total = parts.length
  const skipped = format === 'thunderstore' ? skippedForDependencies(items) : []
  const next = copied >= total ? 0 : copied
  const off = items.filter((item) => !item.enabled).length
  const options = useFormatOptions(thunderstore)
  const count = items.length
  const summary = [
    plural(count, { one: '# mod', other: '# mods' }),
    off > 0 ? plural({ count: off }, { one: '# mod disabled', other: '# mods disabled' }) : '',
    total > 1 ? t`${total} messages` : '',
  ]
    .filter(Boolean)
    .join(' · ')
  const copyOne = (n: number, body: string) =>
    copyText(body, total > 1 ? t`Message ${{ count: n }} of ${total} copied` : t`Mod list copied`)
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
      .catch((error: unknown) => toastError(t`Could not save the file`, error))
  }
  return (
    <Box sx={paneSx}>
      <div className="list-bar">
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
            <ToggleButton key={o.id} value={o.id} sx={{ px: space.pad, gap: space.gap }}>
              <o.icon size={15} />
              {o.label}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
        <p className="list-note">{summary}</p>
      </div>
      {/* A Box, since a scrolling region takes focus to scroll from the keyboard. */}
      <Box role="region" className="list-text" aria-label={t`The text that is copied`} tabIndex={0}>
        {parts.map((part) => (
          <div key={part.id}>
            {total > 1 ? (
              <div className="list-message">
                {part.n <= copied ? <Check size={14} aria-label={t`Copied`} /> : null}
                <p className="list-message-title">
                  {t`Message ${{ count: part.n }} of ${total}`}
                  <span className="list-message-size">
                    {t`${{ count: part.text.length.toLocaleString(i18n.locale) }} characters`}
                  </span>
                </p>
                <TipIconButton
                  label={t`Copy message ${{ count: part.n }}`}
                  onClick={() => copyOne(part.n, part.text)}
                >
                  <Copy size={14} />
                </TipIconButton>
              </div>
            ) : null}
            <pre>{part.text}</pre>
          </div>
        ))}
      </Box>
      <div className="list-actions">
        <ListHint skipped={skipped} total={total} />
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<Save size={16} />}
          onClick={save}
          sx={{ height: 40, px: '14px', fontSize: 'medium.fontSize' }}
        >
          {t`Save to a file…`}
        </Button>
        <Button
          variant="contained"
          startIcon={<Copy size={16} />}
          onClick={copyNext}
          sx={{ height: 40, px: '18px', fontSize: 'medium.fontSize', whiteSpace: 'nowrap' }}
        >
          {total > 1 ? t`Copy message ${{ current: next + 1 }} of ${total}` : t`Copy mod list`}
        </Button>
      </div>
    </Box>
  )
}
