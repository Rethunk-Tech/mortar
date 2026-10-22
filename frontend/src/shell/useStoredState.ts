import { useState } from 'react'

function read<T>(key: string, fallback: T, valid: (value: unknown) => value is T): T {
  try {
    const raw = localStorage.getItem(key)
    if (raw === null) {
      return fallback
    }
    const parsed = parseOrRaw(raw)
    return valid(parsed) ? parsed : fallback
  } catch {
    return fallback
  }
}

// Keys written before they went through this module hold a bare string, which is not JSON.
function parseOrRaw(raw: string): unknown {
  try {
    return JSON.parse(raw)
  } catch {
    return raw
  }
}

function write(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch {
    // Storage can be blocked or full; the choice then lasts until the app closes.
  }
}

// State that survives a restart in localStorage. A stored value that fails `valid` is ignored.
export function useStoredState<T>(
  key: string,
  fallback: T,
  valid: (value: unknown) => value is T,
): [T, (value: T) => void] {
  const [value, setValue] = useState<T>(() => read(key, fallback, valid))
  return [
    value,
    (next) => {
      setValue(next)
      write(key, next)
    },
  ]
}

export { read as readStored, write as writeStored }
