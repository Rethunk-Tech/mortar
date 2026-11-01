import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { ChevronRight, Copy, CornerDownRight, Search } from 'lucide-react'
import { useState } from 'react'
import {
  DecompileMapped,
  GameCode,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { Member } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/dotnet/models.ts'
import { HARMONY, HOOKGEN, MONOMOD } from '../brand/productNames.ts'
import { copyText } from '../share/copyText.ts'
import { MenuAction, MenuNote } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { declares } from './declaration.ts'
import { harmonyPatchSnippet, hookSnippet, memberKey, monoModHookSnippet } from './hook.ts'
import { ilOwner, type SymbolClick } from './symbolAt.ts'

// Members a right-click menu lists, at most.
const MENU_MEMBERS = 6

interface Found {
  owner: string
  members: Member[]
}

// What each right-click resolved to, so the same name on the same line opens its menu at once the next time. The game's
// code does not change while Mortar runs.
const lookups = new Map<string, Found>()
const LOOKUPS_MAX = 500

// Where a right-click landed: the assembly and the type open in the view.
interface Scope {
  game: string
  dir: string
  assembly: string
  type: string
}

// The type that owns a right-clicked name: the open type for its own declaration, else the target IL names on that
// line (read from the IL view, or through the line map from C#), else the open type for a name not after a dot.
// '' when none of those settles it.
async function ownerOf(scope: Scope, at: SymbolClick): Promise<string> {
  if (scope.type !== '' && declares(at.text, at.name)) {
    return scope.type
  }
  if (at.il) {
    return ilOwner(at.text, at.name)
  }
  if (scope.type === '') {
    return ''
  }
  const mapped = await DecompileMapped(scope.game, scope.dir, scope.assembly, scope.type)
  const il = mapped.il.split('\n')
  for (const [i, cs] of (mapped.links ?? []).entries()) {
    const owner = cs === at.line ? ilOwner(il[i] ?? '', at.name) : ''
    if (owner !== '') {
      return owner
    }
  }
  const before = at.text.slice(0, at.text.search(new RegExp(`\\b${at.name}\\b`))).trimEnd()
  return before.endsWith('.') ? '' : scope.type
}

/**
 * What a right-clicked name can do: go to its definition from a use of it, find who uses the member it names, and on a
 * method's own declaration copy a HookGen hook, a MonoMod Hook or a Harmony patch for it, built from the signature the
 * search gives. Only when its owner cannot be told are all the assembly's members by that name offered.
 */
export function SymbolMenu({
  scope,
  at,
  onClose,
  onUsages,
  onDefinition,
}: {
  scope: Scope
  at: SymbolClick | null
  onClose: () => void
  onUsages: (target: string) => void
  onDefinition: (m: Member) => void
}) {
  const { t } = useLingui()
  // The Copy hook row whose submenu is open, and its member.
  const [hooks, setHooks] = useState<{ row: HTMLElement; member: Member } | null>(null)
  const close = () => {
    setHooks(null)
    onClose()
  }
  const name = at?.name ?? ''
  const { game, dir, assembly, type } = scope
  const lookupKey = at ? [game, dir, assembly, type, at.il, at.line, at.name].join('\n') : ''
  const cached = lookups.get(lookupKey)
  const { data: loaded } = useLoaded<Found | null>(
    at === null || cached
      ? null
      : async () => {
          const owner = await ownerOf({ game, dir, assembly, type }, at)
          const hits =
            (await GameCode(game, dir, assembly, owner ? `${owner}::${at.name}` : at.name)) ?? []
          const resolved = {
            owner,
            members: hits.filter((m) => m.method === at.name && (owner === '' || m.type === owner)),
          }
          if (lookups.size >= LOOKUPS_MAX) {
            lookups.clear()
          }
          lookups.set(lookupKey, resolved)
          return resolved
        },
    [lookupKey],
    null,
    reportUnexpected,
  )
  const found = cached ?? loaded
  const members = found?.members.slice(0, MENU_MEMBERS) ?? []
  const onItsDefinition = at !== null && declares(at.text, at.name)
  // A member of another assembly (Unity's, say) is not in this one's search, but its usages can still be found.
  const outside =
    found && found.owner !== '' && members.length === 0 ? `${found.owner}::${name}` : ''
  return (
    <Menu
      open={at !== null}
      onClose={close}
      anchorReference="anchorPosition"
      anchorPosition={at ? { top: at.y, left: at.x } : undefined}
      // Above the full-window dialog it opens from.
      sx={{ zIndex: (theme) => theme.zIndex.modal + 1 }}
    >
      {found === null ? <MenuNote>{t`Looking up ${{ name }}…`}</MenuNote> : null}
      {found !== null && members.length === 0 && outside === '' ? (
        <MenuNote>{t`No member named ${{ name }} in this assembly`}</MenuNote>
      ) : null}
      {outside === '' ? null : (
        <MenuAction
          icon={<Search size={14} />}
          label={t`Find usages of ${{ label: outside }}`}
          onClick={() => onUsages(outside)}
        />
      )}
      {members.flatMap((m) => {
        const key = memberKey(m)
        return [
          // Only when the name could be more than one member does each group say whose it is.
          members.length > 1 ? (
            <MenuNote
              key={`n${key}`}
            >{`${m.type.split('.').at(-1) ?? m.type}::${m.method}`}</MenuNote>
          ) : null,
          onItsDefinition ? null : (
            <MenuAction
              key={`d${key}`}
              icon={<CornerDownRight size={14} />}
              label={t`Go to definition`}
              onClick={() => onDefinition(m)}
            />
          ),
          <MenuAction
            key={`u${key}`}
            icon={<Search size={14} />}
            label={t`Find usages…`}
            onClick={() => onUsages(`${m.type}::${m.method}`)}
          />,
          onItsDefinition && m.kind !== 'field' ? (
            <MenuAction
              key={`h${key}`}
              icon={<Copy size={14} />}
              label={t`Copy hook`}
              expanded={hooks?.member === m}
              end={<ChevronRight size={14} />}
              onClick={(e) => setHooks({ row: e.currentTarget, member: m })}
            />
          ) : null,
        ]
      })}
      <Menu
        open={hooks !== null}
        anchorEl={hooks?.row}
        onClose={() => setHooks(null)}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'left' }}
        sx={{ zIndex: (theme) => theme.zIndex.modal + 2 }}
      >
        {hooks
          ? [
              { id: 'on', text: t`${{ app: HOOKGEN }} On. hook`, make: hookSnippet },
              { id: 'mm', text: t`${{ app: MONOMOD }} Hook`, make: monoModHookSnippet },
              { id: 'hp', text: t`${{ app: HARMONY }} patch`, make: harmonyPatchSnippet },
            ].map((h) => (
              <MenuAction
                key={h.id}
                icon={<Copy size={14} />}
                label={h.text}
                onClick={() => {
                  copyText(h.make(hooks.member), t`Hook copied`)
                  close()
                }}
              />
            ))
          : null}
      </Menu>
    </Menu>
  )
}
