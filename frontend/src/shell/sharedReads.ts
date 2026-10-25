import { GitHubLoggedIn } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { Owner } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/service.ts'
import { ProfileCounts } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { Account } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/itch/service.ts'
import { shareInFlight } from '../profiles/listProfiles.ts'

// Reads that several parts of one screen make at once (a settings search shows the Downloads and Nexus Mods sections
// together, Game select's page and its title-bar menu count profiles together): callers asking while one is under way
// share it.
const owner = shareInFlight<string>()
const github = shareInFlight<boolean>()
const itch = shareInFlight<Awaited<ReturnType<typeof Account>>>()
const counts = shareInFlight<Awaited<ReturnType<typeof ProfileCounts>>>()

/** The app that opens nxm links now, '' when none. */
export const nxmOwner = () => owner('', () => Promise.resolve(Owner()))
export const githubLoggedIn = () => github('', () => Promise.resolve(GitHubLoggedIn()))
export const itchAccount = () => itch('', () => Promise.resolve(Account()))
/** Each game's profile count. */
export const profileCounts = () => counts('', () => Promise.resolve(ProfileCounts()))
