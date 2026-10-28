import { memo, useEffect, useState } from 'react'
import { SearchField } from '../../shell/SearchField.tsx'
import { useDebouncedValue } from '../../shell/useDebouncedValue.ts'

// How long typing pauses before the mods and settings re-filter, so a word typed at speed filters them once.
const SEARCH_SETTLE_MS = 150

// Its own component: only the field follows each key typed; the Config tab hears the query once typing pauses.
export const ConfigSearch = memo(function ConfigSearchView({
  label,
  placeholder,
  onQuery,
}: {
  label: string
  placeholder: string
  onQuery: (query: string) => void
}) {
  const [text, setText] = useState('')
  const settled = useDebouncedValue(text, SEARCH_SETTLE_MS)
  useEffect(() => onQuery(settled), [settled, onQuery])
  return (
    <SearchField
      label={label}
      placeholder={placeholder}
      value={text}
      onChange={setText}
      grow={true}
    />
  )
})
