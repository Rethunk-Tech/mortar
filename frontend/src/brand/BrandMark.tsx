// A brand's vector mark from its 24x24 path, decorative: the control around it carries the name.
export function BrandMark({ path, size, fill }: { path: string; size: number; fill: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" fill={fill}>
      <path d={path} />
    </svg>
  )
}
