import { TourPopover } from './TourPopover.tsx'
import { useFirstRunTour } from './useFirstRunTour.ts'

function FirstRunTour() {
  const { active, anchorEl, step, setStep, finish } = useFirstRunTour()
  if (!(active && anchorEl)) {
    return null
  }
  return <TourPopover anchorEl={anchorEl} step={step} setStep={setStep} finish={finish} />
}

export { FirstRunTour }
