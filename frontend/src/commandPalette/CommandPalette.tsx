import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Dialog,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  TextField,
} from '@mui/material'
import {
  Download,
  FolderOpen,
  FolderPlus,
  Inbox,
  Keyboard,
  Package,
  Play,
  RefreshCw,
  Settings,
  Share2,
  User,
} from 'lucide-react'
import {
  Fragment,
  type KeyboardEvent,
  type ReactNode,
  type RefObject,
  useEffect,
  useRef,
  useState,
} from 'react'
import { BisectDialog } from '../console/BisectDialog.tsx'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import type { SettingsSection } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { shortcutLabels } from '../settings/shortcutLabels.ts'
import { mergeBindings } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { space } from '../theme/density.ts'
import type { PaletteItem } from './match.ts'
import { runPaletteItem } from './run.ts'
import type { PaletteRow, PaletteSection } from './sections.ts'
import { useCommandPalette } from './store.ts'
import { usePaletteShown } from './usePaletteShown.ts'

function iconFor(item: PaletteItem): ReactNode {
  if (item.kind === 'profile') {
    return <User size={16} />
  }
  if (item.kind === 'mod') {
    return <Package size={16} />
  }
  if (item.kind === 'settings') {
    return <Settings size={16} />
  }
  if (item.kind === 'shortcut') {
    return <Keyboard size={16} />
  }
  if (item.id === 'action:play') {
    return <Play size={16} />
  }
  if (item.id === 'shortcut:check-updates') {
    return <RefreshCw size={16} />
  }
  if (item.id === 'action:downloads') {
    return <Download size={16} />
  }
  if (item.id === 'action:downloads-folder') {
    return <FolderOpen size={16} />
  }
  if (item.id === 'action:import') {
    return <Inbox size={16} />
  }
  if (item.id === 'action:share') {
    return <Share2 size={16} />
  }
  return <FolderPlus size={16} />
}

const PALETTE_LIST_ID = 'command-palette-results'
const optionId = (id: string) => `command-palette-option-${id.replace(/[^\w-]/g, '_')}`

function PaletteRows({
  shown,
  currentId,
  onPick,
}: {
  shown: PaletteRow[]
  currentId: string | undefined
  onPick: (id: string) => void
}) {
  const { t } = useLingui()
  const headings: Record<PaletteSection, string> = {
    recent: t`Recent`,
    goto: t`Go to`,
    actions: t`Actions`,
    settings: t`Settings`,
    mods: t`Mods`,
  }
  return (
    <List
      id={PALETTE_LIST_ID}
      dense={true}
      sx={{ height: '60vh', overflow: 'auto', py: space.gap }}
      role="listbox"
      aria-label={t`Results`}
    >
      {shown.map(({ item, section }, at) => (
        <Fragment key={item.id}>
          {section === shown[at - 1]?.section ? null : (
            <Box
              role="presentation"
              sx={{
                px: space.pad,
                pt: at === 0 ? 0 : 1,
                pb: 0.5,
                fontSize: 11,
                fontWeight: 700,
                letterSpacing: '0.08em',
                textTransform: 'uppercase',
                color: 'var(--mortar-ink-sec)',
              }}
            >
              {headings[section]}
            </Box>
          )}
          <ListItemButton
            id={optionId(item.id)}
            // The search field keeps focus and arrow keys move the selection, so options are not Tab stops.
            tabIndex={-1}
            selected={item.id === currentId}
            onClick={() => onPick(item.id)}
            disabled={item.disabled !== undefined}
            role="option"
            aria-selected={item.id === currentId}
          >
            <ListItemIcon sx={{ minWidth: 32, color: 'var(--mortar-ink-sec)' }}>
              {iconFor(item)}
            </ListItemIcon>
            <ListItemText
              primary={item.label}
              secondary={item.hint}
              slotProps={{ primary: { noWrap: true }, secondary: { noWrap: true } }}
            />
            {item.shortcut ? (
              <kbd style={{ color: 'var(--mortar-ink-sec)', fontSize: 12 }}>{item.shortcut}</kbd>
            ) : null}
          </ListItemButton>
        </Fragment>
      ))}
    </List>
  )
}

function usePaletteWindow(open: boolean) {
  const searchRef = useRef<HTMLInputElement>(null)
  useEffect(() => {
    if (!open) {
      return
    }
    const focus = requestAnimationFrame(() => {
      searchRef.current?.focus()
    })
    const onEsc = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        useCommandPalette.getState().setOpen(false)
      }
    }
    globalThis.addEventListener('keydown', onEsc, true)
    return () => {
      cancelAnimationFrame(focus)
      globalThis.removeEventListener('keydown', onEsc, true)
    }
  }, [open])
  return searchRef
}

function paletteSections(i18n: I18n) {
  return [
    { id: 'general' as const, label: i18n._(msg`General`) },
    { id: 'appearance' as const, label: i18n._(msg`Appearance`) },
    { id: 'mods' as const, label: i18n._(msg`Mods and profiles`) },
    { id: 'downloads' as const, label: i18n._(msg`Downloads`) },
    { id: 'accounts' as const, label: i18n._(msg`Accounts`) },
    { id: 'sources' as const, label: i18n._(msg`Source health`) },
    { id: 'updates' as const, label: i18n._(msg`Updates`) },
    { id: 'notifications' as const, label: i18n._(msg`Notifications`) },
    { id: 'storage' as const, label: i18n._(msg`Storage`) },
    { id: 'launchers' as const, label: i18n._(msg`Launchers`) },
    { id: 'shortcuts' as const, label: i18n._(msg`Shortcuts`) },
    { id: 'about' as const, label: i18n._(msg`About`) },
  ]
}

const closePalette = () => useCommandPalette.getState().setOpen(false)

function PaletteBody({ searchRef }: { searchRef: RefObject<HTMLInputElement | null> }) {
  const { t, i18n } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const openId = useProfiles((s) => s.openId)
  const game = useProfiles((s) => s.game)
  const [query, setQuery] = useState('')
  const [index, setIndex] = useState(0)
  const sections: { id: SettingsSection; label: string }[] = paletteSections(i18n)
  const shortcutLabelMap = shortcutLabels(i18n)
  const bindings = mergeBindings(useSettings((s) => s.shortcuts))
  const rows = usePaletteShown({
    i18n,
    query,
    profiles,
    openId,
    gameId: game?.id ?? '',
    sections,
    shortcutLabels: shortcutLabelMap,
    bindings,
  })
  const shown = rows.map((row) => row.item)
  const current = shown[Math.min(index, Math.max(shown.length - 1, 0))]
  const pick = (id: string) => {
    if (shown.find((item) => item.id === id)?.disabled !== undefined) {
      return
    }
    runPaletteItem(id)
    closePalette()
  }
  const onKey = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      closePalette()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setIndex((i) => (shown.length === 0 ? 0 : (i + 1) % shown.length))
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      setIndex((i) => (shown.length === 0 ? 0 : (i - 1 + shown.length) % shown.length))
      return
    }
    if (e.key === 'Enter' && current) {
      e.preventDefault()
      pick(current.id)
    }
  }
  return (
    <>
      <TextField
        inputRef={searchRef}
        autoFocus={true}
        fullWidth={true}
        value={query}
        onChange={(e) => {
          setQuery(e.target.value)
          setIndex(0)
        }}
        onKeyDown={onKey}
        placeholder={t`Search actions and destinations`}
        slotProps={{
          htmlInput: {
            'aria-label': t`Search actions and destinations`,
            role: 'combobox',
            'aria-expanded': shown.length > 0,
            'aria-controls': PALETTE_LIST_ID,
            'aria-autocomplete': 'list',
            'aria-activedescendant': current ? optionId(current.id) : undefined,
          },
        }}
        sx={{ px: space.pad, pt: space.pad, '& .MuiInputBase-root': { userSelect: 'text' } }}
      />
      <PaletteRows shown={rows} currentId={current?.id} onPick={pick} />
    </>
  )
}

export function CommandPalette() {
  const { t } = useLingui()
  const open = useCommandPalette((s) => s.open)
  const creating = useCommandPalette((s) => s.creating)
  const bisect = useCommandPalette((s) => s.bisect)
  const searchRef = usePaletteWindow(open)
  // The body subscribes to profiles and settings; a closed palette renders none of it, and its query starts empty
  // each time it opens.
  return (
    <>
      <Dialog
        open={open}
        onClose={closePalette}
        slotProps={{
          paper: {
            'aria-label': t`Command palette`,
            sx: { width: 560, maxWidth: 'calc(100% - 48px)' },
          },
        }}
      >
        <PaletteBody searchRef={searchRef} />
      </Dialog>
      <NewProfileDialog
        open={creating}
        onClose={() => useCommandPalette.getState().setCreating(false)}
      />
      {bisect ? (
        <BisectDialog
          game={bisect.game}
          profile={bisect.profile}
          jobID={bisect.id}
          onClose={() => useCommandPalette.getState().setBisect(null)}
        />
      ) : null}
    </>
  )
}
