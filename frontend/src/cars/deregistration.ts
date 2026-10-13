import type { Car } from '#/api/client'
import { toCalendarDate } from '#/lib/date-range'

// A car is deregistered once its deregistration date is today or earlier in
// the viewer's time zone, so a future date takes effect at local midnight.
// Full-dates order lexically, so they compare as strings.
export function isDeregistered(
  car: Pick<Car, 'deregistrationDate'>,
  today: Date = new Date(),
): boolean {
  return (
    car.deregistrationDate !== null &&
    car.deregistrationDate <= toCalendarDate(today)
  )
}

// Rows of an inactive or deregistered car are shown muted; isActive and the
// deregistration date are independent of each other.
export function isCarMuted(
  car: Pick<Car, 'deregistrationDate' | 'isActive'>,
  today: Date = new Date(),
): boolean {
  return !car.isActive || isDeregistered(car, today)
}
