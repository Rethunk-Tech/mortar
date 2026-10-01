declare module '*internal/bisect/service.js' {
  export function Start(gameID: string, profileID: string): Promise<string>
  export function Status(id: string): Promise<{
    state: string
    step: number
    total: number
    modsLeft: number
    result?: {
      mods: Array<{
        key: string
        uniqueId: string
        name: string
      }>
    }
    error?: string
  }>
  export function Stop(id: string): Promise<void>
  export function SwitchOff(
    gameID: string,
    profileID: string,
    key: string,
    uniqueID: string,
  ): Promise<void>
}
