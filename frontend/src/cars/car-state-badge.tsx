import { useTranslation } from 'react-i18next'

import type { Car } from '#/api/client'

import { isCarMuted, isDeregistered } from './deregistration'

type CarState = Pick<Car, 'deregistrationDate' | 'isActive'>

// The class of a table row belonging to car, which may be unknown.
export function carRowClass(car: CarState | undefined): string | undefined {
  return car !== undefined && isCarMuted(car) ? 'row-muted' : undefined
}

// The text badge on a muted row; deregistration wins over inactivity when
// both apply, since it is the stronger state.
export function CarStateBadge({ car }: { car: CarState | undefined }) {
  const { t } = useTranslation()
  if (car === undefined) return null
  const state = isDeregistered(car)
    ? 'deregistered'
    : car.isActive
      ? undefined
      : 'inactive'
  if (state === undefined) return null
  return <span className="car-state-badge">{t(`cars.states.${state}`)}</span>
}
