import type { Peer } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/models.ts'

export function peerSecondary(
  peer: Pick<Peer, 'id' | 'paired'>,
  sharedName: boolean,
  paired: string,
  unpaired: string,
): string {
  const status = peer.paired ? paired : unpaired
  return sharedName ? `${peer.id} · ${status}` : status
}
