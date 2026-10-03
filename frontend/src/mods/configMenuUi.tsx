import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Link, Typography } from '@mui/material'
import {
  type GmcmCapture,
  type GmcmOption,
  type GmcmPage,
  type GmcmResult,
  optionKey,
  valuesEqual,
} from './configMenu.ts'
import { MenuControl } from './configMenuControls.tsx'
import { menuControl } from './configMenuKinds.ts'

const text = { fontSize: 13 } as const

function asLinkTarget(opt: GmcmOption): string {
  if (typeof opt.value === 'string' && opt.value) {
    return opt.value
  }
  return opt.name
}

function OptionRow({
  page,
  opt,
  drafts,
  onChange,
  onPage,
}: {
  page: GmcmPage
  opt: GmcmOption
  drafts: Record<string, unknown>
  onChange: (key: string, value: unknown) => void
  onPage: (id: string) => void
}) {
  const { t, i18n } = useLingui()
  const kind = menuControl(opt.kind)
  const pending =
    optionKey(page.id, opt.index) in drafts &&
    !valuesEqual(drafts[optionKey(page.id, opt.index)], opt.value)
  if (kind === 'section') {
    return <Typography variant="subtitle1">{opt.name}</Typography>
  }
  if (kind === 'subHeader') {
    return <Typography variant="subtitle2">{opt.name}</Typography>
  }
  if (kind === 'paragraph') {
    return <Typography sx={text}>{opt.name}</Typography>
  }
  if (kind === 'pageLink') {
    return (
      <Link component="button" type="button" onClick={() => onPage(asLinkTarget(opt))} sx={text}>
        {opt.name}
      </Link>
    )
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography sx={text}>{opt.name}</Typography>
        {pending ? <Chip size="small" label={t`Pending`} /> : null}
      </Box>
      {opt.tooltip ? (
        <Typography sx={{ ...text, color: 'text.secondary' }}>{opt.tooltip}</Typography>
      ) : null}
      <MenuControl page={page.id} opt={opt} drafts={drafts} i18n={i18n} onChange={onChange} />
    </Box>
  )
}

export function MenuPages({
  capture,
  pageId,
  drafts,
  result,
  onPage,
  onChange,
  onDiscard,
}: {
  capture: GmcmCapture
  pageId: string
  drafts: Record<string, unknown>
  result: GmcmResult | null
  onPage: (id: string) => void
  onChange: (key: string, value: unknown) => void
  onDiscard: () => void
}) {
  const { t } = useLingui()
  const page = capture.pages.find((p) => p.id === pageId) ?? capture.pages[0]
  const skipped = result?.skipped ?? []
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Typography sx={text}>{t`Applies next time the game starts`}</Typography>
      {page ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <Typography variant="subtitle1">{page.title || page.id}</Typography>
          {(page.options ?? []).map((opt) => (
            <OptionRow
              key={optionKey(page.id, opt.index)}
              page={page}
              opt={opt}
              drafts={drafts}
              onChange={onChange}
              onPage={onPage}
            />
          ))}
        </Box>
      ) : null}
      {Object.keys(drafts).length > 0 ? (
        <Button onClick={onDiscard} sx={{ alignSelf: 'flex-start', whiteSpace: 'nowrap' }}>
          {t`Discard pending`}
        </Button>
      ) : null}
      {skipped.map((row) => (
        <Typography key={`${row.edit.page}/${row.edit.index}/${row.reason}`} sx={text}>
          {`${row.edit.name}: ${row.reason}`}
        </Typography>
      ))}
    </Box>
  )
}

export function MenuHint() {
  const { t } = useLingui()
  return (
    <Typography
      sx={text}
    >{t`Start the game once with this profile to load this mod's settings menu.`}</Typography>
  )
}
