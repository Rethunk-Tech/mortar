import { TourPopover } from './TourPopover.tsx'
import { useFirstRunTour } from './useFirstRunTour.ts'

function FirstRunTour() {
  const { active, anchorEl, anchorRect, step, setStep, finish } = useFirstRunTour()
  if (!(active && anchorEl && anchorRect)) {
    return null
  }
  return (
    <TourPopover
      anchorEl={anchorEl}
      anchorRect={anchorRect}
      step={step}
      setStep={setStep}
      finish={finish}
    />
  )
}

export { FirstRunTour }
