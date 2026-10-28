import { useLingui } from '@lingui/react/macro'
import {
  Autocomplete,
  Box,
  Button,
  CircularProgress,
  Drawer,
  MenuItem,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material'
import { CircleAlert, Code, Copy, FileSearch, Search } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import type {
  Mapped,
  Usage,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Assemblies,
  Decompile,
  DecompileMapped,
  GameCode,
  HasDecompiler,
  InstallDecompiler,
  Usages,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { Member } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/dotnet/models.ts'
import { ILSPY, MONOMOD } from '../brand/productNames.ts'
import { copyText } from '../share/copyText.ts'
import { AppDialog, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { labelledInput } from '../shell/labelledInput.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { useDebouncedLoad } from '../shell/useDebouncedLoad.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import {
  errorMessage,
  type InlineError,
  inlineError,
  reportError,
  reportUnexpected,
} from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { CodeView } from './CodeView.tsx'
import { hookSnippet } from './hook.ts'
import { MappedView } from './MappedView.tsx'

// A constructor's decompiled declaration is named after its type, which a search for ".ctor" cannot find.
const CONSTRUCTOR_NAME = /^\.c?ctor$/
const TYPE_PARTS = /[.+]/

// Typing settles for this long before the game's members are searched.
const SEARCH_DELAY_MS = 200
const SEARCH_PX = 420
const ASSEMBLY_PX = 240
const USAGES_PX = 420
const CSHARP = 'C#'
const IL = 'IL'

type CodeMode = 'cs' | 'il' | 'both'

function useHits(game: string, dir: string, assembly: string, query: string) {
  const searching = query.trim().length >= 2
  const { data, error } = useDebouncedLoad<Member[]>(
    searching ? () => GameCode(game, dir, assembly, query).then((found) => found ?? []) : null,
    [game, dir, assembly, query, searching],
    { initial: [], delayMs: SEARCH_DELAY_MS },
  )
  return { hits: data, error: error === null ? '' : errorMessage(error) }
}

// The line declaring method in decompiled source: its name before a parameter list, on a line that is not a call.
function declarationLine(source: string, type: string, method: string): number {
  const name = CONSTRUCTOR_NAME.test(method) ? (type.split(TYPE_PARTS).at(-1) ?? '') : method
  return (
    source
      .split('\n')
      // IL writes a space before the parameter list.
      .findIndex(
        (line) =>
          (line.includes(` ${name}(`) || line.includes(` ${name} (`)) &&
          !line.trimEnd().endsWith(';'),
      )
  )
}

function Source({
  game,
  dir,
  assembly,
  type,
  method,
  mode,
}: {
  game: string
  dir: string
  assembly: string
  type: string
  method: string
  mode: CodeMode
}) {
  const { t } = useLingui()
  const [has, setHas] = useState<boolean | null>(null)
  const [source, setSource] = useState('')
  const [mapped, setMapped] = useState<Mapped | null>(null)
  const [failure, setFailure] = useState<InlineError | null>(null)
  const il = mode === 'il'
  const [installing, install] = usePending()
  useEffect(() => {
    HasDecompiler()
      .then(setHas)
      .catch(() => setHas(false))
  }, [])
  const load = useCallback(() => {
    if (!has || type === '') {
      return
    }
    setSource('')
    setMapped(null)
    setFailure(null)
    const failed = (e: unknown) => setFailure(inlineError(e))
    if (mode === 'both') {
      DecompileMapped(game, dir, assembly, type).then(setMapped).catch(failed)
    } else {
      Decompile(game, dir, assembly, type, mode === 'il')
        .then(setSource)
        .catch(failed)
    }
  }, [game, dir, assembly, type, mode, has])
  useEffect(load, [load])
  if (has === false) {
    return (
      <EmptyState
        icon={<Code size={40} aria-hidden={true} />}
        title={t`Install ${{ loader: ILSPY }} to read the game's code`}
        action={
          <Button
            variant="contained"
            disabled={installing}
            onClick={() =>
              install(
                async () => {
                  await InstallDecompiler()
                  setHas(await HasDecompiler())
                },
                { errorTitle: t`Could not install ${{ loader: ILSPY }}` },
              )
            }
          >
            {installing ? t`Installing…` : t`Install ${{ loader: ILSPY }}`}
          </Button>
        }
      >
        {t`Mortar installs its command line as a .NET tool and decompiles the type you pick.`}
      </EmptyState>
    )
  }
  if (type === '') {
    return (
      <EmptyState icon={<FileSearch size={40} aria-hidden={true} />} title={t`Pick a method`}>
        {t`Search the game's types and methods, then pick one to read its type's code.`}
      </EmptyState>
    )
  }
  if (failure) {
    return (
      <EmptyState
        icon={<CircleAlert size={40} aria-hidden={true} />}
        title={t`Could not decompile ${{ type }}`}
        action={
          <Button variant="outlined" onClick={load}>
            {t`Try again`}
          </Button>
        }
      >
        {failure.details || failure.message}
      </EmptyState>
    )
  }
  if (mode === 'both' && mapped) {
    return <MappedView mapped={mapped} line={declarationLine(mapped.csharp, type, method)} />
  }
  if (mode === 'both' || source === '') {
    return (
      <EmptyState
        icon={<CircularProgress size={36} aria-hidden={true} />}
        title={mode === 'both' ? t`Matching C# to IL` : t`Decompiling ${{ type }}`}
      >
        {mode === 'both' ? t`The first type of an assembly takes a while.` : ''}
      </EmptyState>
    )
  }
  return <CodeView source={source} il={il} line={declarationLine(source, type, method)} />
}

// The methods, in every assembly the view lists, that use the picked member; picking one opens it.
function UsagesDrawer({
  game,
  dir,
  target,
  open,
  onClose,
  onOpen,
}: {
  game: string
  dir: string
  target: string
  open: boolean
  onClose: () => void
  onOpen: (u: Usage) => void
}) {
  const { t } = useLingui()
  const { data: usages } = useLoaded<Usage[] | null>(
    open && target !== '' ? () => Usages(game, dir, target) : null,
    [open, game, dir, target],
    null,
    reportError(t`Could not find usages`),
  )
  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      // Above the full-window dialog it opens from, on the dialog's own solid surface.
      sx={{ zIndex: (theme) => theme.zIndex.modal + 1 }}
      slotProps={{
        paper: {
          sx: { bgcolor: 'var(--mortar-raised)', borderLeft: '1px solid var(--mortar-hairline)' },
        },
      }}
    >
      <Box
        sx={{
          width: USAGES_PX,
          p: space.pad,
          display: 'flex',
          flexDirection: 'column',
          gap: space.gap,
        }}
      >
        <Typography sx={{ fontWeight: 600 }}>{t`Usages`}</Typography>
        <Typography noWrap={true} sx={{ fontFamily: MONO, fontSize: 'small.fontSize' }}>
          {target}
        </Typography>
        {usages === null ? <LoadingRow>{t`Finding usages…`}</LoadingRow> : null}
        {usages?.length === 0 ? (
          <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
            {t`Nothing in these assemblies uses it.`}
          </Typography>
        ) : null}
        {usages?.map((u) => (
          <Box
            key={`${u.assembly}|${u.caller}`}
            component="button"
            type="button"
            onClick={() => onOpen(u)}
            sx={{ all: 'unset', display: 'block', cursor: 'pointer', py: 0.25 }}
          >
            <Typography noWrap={true} sx={{ fontFamily: MONO, fontSize: 'small.fontSize' }}>
              {u.caller}
            </Typography>
            <Typography noWrap={true} sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
              {u.label}
            </Typography>
          </Box>
        ))}
      </Box>
    </Drawer>
  )
}

const memberKey = (m: Member) => `${m.type}::${m.method}(${(m.params ?? []).join(',')})`

// The search: hits drop down under it, a type or method on each; picking one opens it.
function MemberSearch({
  game,
  dir,
  assembly,
  picked,
  onPick,
}: {
  game: string
  dir: string
  assembly: string
  picked: Member | null
  onPick: (m: Member) => void
}) {
  const { t } = useLingui()
  const [query, setQuery] = useState('')
  const { hits, error } = useHits(game, dir, assembly, query)
  return (
    <Autocomplete
      size="small"
      sx={{ width: SEARCH_PX }}
      options={hits}
      value={picked}
      inputValue={query}
      filterOptions={(all) => all}
      isOptionEqualToValue={(a, b) => memberKey(a) === memberKey(b)}
      getOptionLabel={(m) => `${m.type}::${m.method}`}
      getOptionKey={memberKey}
      noOptionsText={error || t`Type two letters or more`}
      onInputChange={(_, next, reason) => reason === 'input' && setQuery(next)}
      onChange={(_, m) => m && onPick(m)}
      renderOption={({ key, ...props }, m) => (
        <Box component="li" key={key} {...props} sx={{ display: 'block !important' }}>
          <Typography noWrap={true} sx={{ fontFamily: MONO, fontSize: 'small.fontSize' }}>
            {m.kind === 'field' ? m.method : `${m.method}(${(m.params ?? []).join(', ')})`}
          </Typography>
          <Typography noWrap={true} sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
            {m.type}
          </Typography>
        </Box>
      )}
      renderInput={labelledInput(
        t`Search the game's code`,
        t`Type or method, such as player update`,
      )}
    />
  )
}

/** Search the game's code, read a type decompiled as C#, IL or both side by side, and copy a hook for a method. */
export function GameCodeDialog({
  game,
  dir,
  onClose,
}: {
  game: string
  dir: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const [picked, setPicked] = useState<Member | null>(null)
  const [assembly, setAssembly] = useState('')
  const [mode, setMode] = useState<CodeMode>('cs')
  const [usages, setUsages] = useState(false)
  const assemblies =
    useLoaded(() => Assemblies(game, dir), [game, dir], null, reportUnexpected).data ?? []
  const target = picked ? `${picked.type}::${picked.method}` : ''
  const actions = (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, flexWrap: 'wrap' }}>
      <TextField
        select={true}
        size="small"
        value={assembly}
        onChange={(e) => {
          setAssembly(e.target.value)
          setPicked(null)
        }}
        slotProps={{ select: { displayEmpty: true }, htmlInput: { 'aria-label': t`Assembly` } }}
        sx={{ width: ASSEMBLY_PX }}
      >
        {assemblies.map((a, i) => (
          <MenuItem key={a.path} value={i === 0 && a.group === 'game' ? '' : a.path}>
            {a.label}
          </MenuItem>
        ))}
      </TextField>
      <MemberSearch game={game} dir={dir} assembly={assembly} picked={picked} onPick={setPicked} />
      <ToggleButtonGroup
        exclusive={true}
        size="small"
        value={mode}
        onChange={(_, next: CodeMode | null) => next && setMode(next)}
        aria-label={t`Code view`}
      >
        <ToggleButton value="cs">{CSHARP}</ToggleButton>
        <ToggleButton value="il">{IL}</ToggleButton>
        <ToggleButton value="both">{t`Side by side`}</ToggleButton>
      </ToggleButtonGroup>
      <Button
        variant="outlined"
        startIcon={<Search size={14} />}
        disabled={target === ''}
        onClick={() => setUsages(true)}
      >
        {t`Find usages`}
      </Button>
      <TipIconButton
        label={t`Copy a ${{ library: MONOMOD }} hook`}
        size="medium"
        disabled={!picked || picked.kind === 'field'}
        onClick={() => picked && copyText(hookSnippet(picked), t`Hook copied`)}
      >
        <Copy size={16} />
      </TipIconButton>
    </Box>
  )
  return (
    <AppDialog
      open={true}
      onClose={onClose}
      title={t`Game code`}
      icon={
        <ToneIcon tone="neutral">
          <Code size={22} />
        </ToneIcon>
      }
      size="full"
      flush={true}
      actions={actions}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
        <Source
          game={game}
          dir={dir}
          assembly={assembly}
          type={picked?.type ?? ''}
          method={picked?.method ?? ''}
          mode={mode}
        />
      </Box>
      <UsagesDrawer
        game={game}
        dir={dir}
        target={target}
        open={usages}
        onClose={() => setUsages(false)}
        onOpen={(u) => {
          const [type = '', method = ''] = u.caller.split('::')
          setAssembly(u.assembly === assemblies[0]?.path ? '' : u.assembly)
          setPicked({ type, kind: 'method', method, static: false, void: true, params: [] })
          setUsages(false)
        }}
      />
    </AppDialog>
  )
}
