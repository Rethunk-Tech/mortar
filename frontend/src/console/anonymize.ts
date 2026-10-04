const homePath = /\/home\/([^/\\\s]+)/

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// Hides the OS user name found in a /home/<user> path: the path becomes ~ and any other mention becomes <user>.
export function anonymize(log: string): string {
  const user = log.match(homePath)?.[1]
  if (!user) {
    return log
  }
  return log
    .replaceAll(`/home/${user}`, '~')
    .replaceAll(new RegExp(`(?<!\\w)${escapeRegExp(user)}(?!\\w)`, 'g'), '<user>')
}
