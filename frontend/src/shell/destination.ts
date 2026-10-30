import { readStored, writeStored } from '@rethunk/react-load-hooks'

// Redirects on Mortar's site to Discord's permanent invites, so a moved channel needs only a site change.
export const GO = 'https://mortar.rethunk.tech/go/'

export type Destination = 'discord' | 'github'

export const REPORT_KEY = 'mortar.reportDestination'
export const SUGGEST_KEY = 'mortar.suggestDestination'

const isDestination = (v: unknown): v is Destination => v === 'discord' || v === 'github'

export const getDestination = (key: string): Destination => readStored(key, 'github', isDestination)

export const setDestination = (key: string, value: Destination): void => writeStored(key, value)
