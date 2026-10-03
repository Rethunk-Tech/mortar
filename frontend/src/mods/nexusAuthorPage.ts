const usersPath = /nexusmods\.com\/users\/(\d+)/i

export function nexusAuthorPageUrl(uploaderUrl: string, modPageUrl: string): string {
  const match = uploaderUrl.match(usersPath)
  if (match?.[1]) {
    return `https://www.nexusmods.com/users/${match[1]}`
  }
  if (uploaderUrl.trim() !== '') {
    return uploaderUrl
  }
  return modPageUrl
}
