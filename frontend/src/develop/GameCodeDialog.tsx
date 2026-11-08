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
import { useDebouncedLoad, useLoaded } from '@rethunk/react-load-hooks'
import { CircleAlert, Code, FileSearch } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Member } from '../../bindings/github.com/Rethunk-Tech/go-dotnet/models.ts'
import type {
  Assembly,
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
import { ILSPY } from '../brand/productNames.ts'
import type { StackFrame } from '../console/stackFrame.ts'
import { AppDialog, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { labelledInput } from '../shell/labelledInput.tsx'
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
import { declarationLine } from './declaration.ts'
import { memberKey } from './hook.ts'
import { MappedView } from './MappedView.tsx'
import { SymbolMenu } from './SymbolMenu.tsx'
import type { SymbolClick } from './symbolAt.ts'
import { TraceSteps } from './TraceSteps.tsx'

// A constructor's decompiled declaration is named after its type, which a search for ".ctor" cannot find.

// Typing settles for this long before the game's members are searched.
const SEARCH_DELAY_MS = 200
const SEARCH_PX = 420
const ASSEMBLY_PX = 240
// The narrowest the assembly and member pickers get before the header has no room left.
const PICKER_MIN_PX = 140
const USAGES_PX = 420
const CSHARP = 'C#'
const IL_LABEL = 'IL'

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

function Source({
  game,
  dir,
  assembly,
  type,
  kind,
  method,
  mode,
  onSymbol,
}: {
  game: string
  dir: string
  assembly: string
  type: string
  kind: string
  method: string
  mode: CodeMode
  onSymbol: (at: SymbolClick) => void
}) {
  const { t } = useLingui()
  const [has, setHas] = useState<boolean | null>(null)
  const [source, setSource] = useState('')
  const [mapped, setMapped] = useState<Mapped | null>(null)
  const [failure, setFailure] = useState<InlineError | null>(null)
  const il = mode === 'il'
  const [installing, install] = usePending()
  // Counts loads, so only the newest one's result lands.
  const latest = useRef(0)
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
    latest.current += 1
    const id = latest.current
    const live =
      <T,>(set: (v: T) => void) =>
      (v: T) =>
        id === latest.current && set(v)
    const failed = live((e: unknown) => setFailure(inlineError(e)))
    if (mode === 'both') {
      DecompileMapped(game, dir, assembly, type).then(live(setMapped)).catch(failed)
    } else {
      Decompile(game, dir, assembly, type, mode === 'il')
        .then(live(setSource))
        .catch(failed)
    }
  }, [game, dir, assembly, type, mode, has])
  useEffect(() => {
    load()
    return () => {
      latest.current += 1
    }
  }, [load])
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
    return <MappedView mapped={mapped} member={{ type, kind, name: method }} onSymbol={onSymbol} />
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
  return (
    <SingleView
      key={`${type}\n${kind}\n${method}\n${source.length}`}
      source={source}
      il={il}
      line={declarationLine(source, type, kind, method)}
      onSymbol={onSymbol}
    />
  )
}

// One side alone tints the opened member's declaration, then whichever line is clicked, as Side by side does.
function SingleView({
  source,
  il,
  line,
  onSymbol,
}: {
  source: string
  il: boolean
  line: number
  onSymbol: (at: SymbolClick) => void
}) {
  const [picked, setPicked] = useState(line)
  const marked = useMemo(() => new Set(picked >= 0 ? [picked] : []), [picked])
  return (
    <CodeView
      source={source}
      il={il}
      line={line}
      marked={marked}
      onPick={setPicked}
      onSymbol={onSymbol}
    />
  )
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
            sx={{
              all: 'unset',
              display: 'block',
              cursor: 'pointer',
              py: 0.25,
              '&:focus-visible': { outline: '2px solid', outlineColor: 'primary.main' },
            }}
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
  // A pick from the list keeps what was typed; one made elsewhere (a trace, Usages) shows its name instead.
  const ownPick = useRef(false)
  const { hits, error } = useHits(game, dir, assembly, query)
  return (
    <Autocomplete
      size="small"
      sx={{ flex: `0 1 ${SEARCH_PX}px`, minWidth: PICKER_MIN_PX }}
      options={hits}
      value={picked}
      inputValue={query}
      filterOptions={(all) => all}
      isOptionEqualToValue={(a, b) => memberKey(a) === memberKey(b)}
      getOptionLabel={(m) => `${m.type}::${m.method}`}
      getOptionKey={memberKey}
      noOptionsText={error || t`Type two letters or more`}
      onInputChange={(_, next, reason) => {
        if (reason === 'input' || (reason === 'reset' && !ownPick.current && next !== '')) {
          setQuery(next)
        }
        ownPick.current = false
      }}
      onChange={(_, m) => {
        ownPick.current = true
        if (m) {
          onPick(m)
        }
      }}
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

// The same empty list each render while the list loads, so what depends on it stays put.
const NO_ASSEMBLIES: Assembly[] = []

/** Search the game's code, read a type decompiled as C#, IL or both side by side, and copy a hook for a method. Opened
 * from a Console stack trace, it steps through the trace's frames. */
export function GameCodeDialog({
  game,
  dir,
  trace,
  onClose,
}: {
  game: string
  dir: string
  trace?: { frames: StackFrame[]; at: number }
  onClose: () => void
}) {
  const { t } = useLingui()
  const [picked, setPicked] = useState<Member | null>(null)
  const [assembly, setAssembly] = useState('')
  const [mode, setMode] = useState<CodeMode>('cs')
  // The member whose usages the drawer lists, as "Type::Member"; '' while it is closed.
  const [usagesOf, setUsagesOf] = useState('')
  const [symbol, setSymbol] = useState<SymbolClick | null>(null)
  const assemblies =
    useLoaded(() => Assemblies(game, dir), [game, dir], null, reportUnexpected).data ??
    NO_ASSEMBLIES
  const main = assemblies[0]?.path
  const openFrame = useCallback(
    (from: string, m: Member) => {
      setAssembly(from === main ? '' : from)
      setPicked(m)
    },
    [main],
  )
  // The pickers shrink before the title wraps; the view buttons keep their width.
  const actions = (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        flex: '1 1 auto',
        minWidth: 0,
        justifyContent: 'flex-end',
      }}
    >
      {trace ? <TraceSteps {...{ game, dir, assemblies, trace }} onOpen={openFrame} /> : null}
      <TextField
        select={true}
        size="small"
        value={assembly}
        onChange={(e) => {
          setAssembly(e.target.value)
          setPicked(null)
        }}
        slotProps={{ select: { displayEmpty: true }, htmlInput: { 'aria-label': t`Assembly` } }}
        sx={{ flex: `0 1 ${ASSEMBLY_PX}px`, minWidth: PICKER_MIN_PX }}
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
        sx={{ flexShrink: 0 }}
      >
        <ToggleButton value="cs">{CSHARP}</ToggleButton>
        <ToggleButton value="il">{IL_LABEL}</ToggleButton>
        <ToggleButton value="both">{t`Side by side`}</ToggleButton>
      </ToggleButtonGroup>
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
          kind={picked?.kind ?? 'method'}
          method={picked?.method ?? ''}
          mode={mode}
          onSymbol={setSymbol}
        />
      </Box>
      <UsagesDrawer
        game={game}
        dir={dir}
        target={usagesOf}
        open={usagesOf !== ''}
        onClose={() => setUsagesOf('')}
        onOpen={(u) => {
          const [type = '', method = ''] = u.caller.split('::')
          setAssembly(u.assembly === assemblies[0]?.path ? '' : u.assembly)
          setPicked({
            type,
            kind: 'method',
            method,
            static: false,
            void: true,
            params: [],
            types: [],
            returns: '',
          })
          setUsagesOf('')
        }}
      />
      <SymbolMenu
        scope={{ game, dir, assembly, type: picked?.type ?? '' }}
        at={symbol}
        onClose={() => setSymbol(null)}
        onUsages={(target) => {
          setSymbol(null)
          setUsagesOf(target)
        }}
        onDefinition={(m) => {
          setSymbol(null)
          setPicked(m)
        }}
      />
    </AppDialog>
  )
}
