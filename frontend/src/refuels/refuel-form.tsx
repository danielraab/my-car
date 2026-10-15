import { LoaderCircle, Save } from 'lucide-react'
import { type FormEvent, type ReactNode, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  ApiError,
  type Car,
  type Refuel,
  type RefuelFuel,
  type RefuelInput,
  refuelFuels,
} from '#/api/client'
import { InputSuffix } from '#/components/input-suffix'
import { useUnitFormat } from '#/lib/display-units'
import { fromLocalInput, toLocalInput } from '#/repairs/format'

type Values = {
  carId: string
  date: string
  station: string
  odometerReading: string
  fuel: RefuelFuel
  liters: string
  amount: string
}

type FieldName = keyof Values

const fieldNames: FieldName[] = [
  'carId',
  'date',
  'station',
  'odometerReading',
  'fuel',
  'liters',
  'amount',
]

const knownReasons = new Set([
  'required',
  'empty',
  'invalid_enum',
  'invalid_decimal',
  'negative',
  'non_positive',
  'invalid_type',
])

function initialValues(refuel: Refuel | undefined, carId: string): Values {
  return {
    carId: refuel?.carId ?? carId,
    date: toLocalInput(refuel ? new Date(refuel.date) : new Date()),
    station: refuel?.station ?? '',
    odometerReading:
      refuel?.odometerReading == null ? '' : String(refuel.odometerReading),
    fuel: refuel?.fuel ?? refuelFuels[0],
    liters: refuel?.liters ?? '',
    amount: refuel?.amount ?? '',
  }
}

// Accept a decimal comma ("42,5"), as German users write amounts.
function decimal(value: string): string {
  const trimmed = value.trim()
  return /^-?\d+,\d+$/.test(trimmed) ? trimmed.replace(',', '.') : trimmed
}

// A whole number is sent as one (the backend reports a negative one); any
// other non-empty text cannot be sent as an integer and is rejected here.
function odometer(value: string): number | null | 'invalid' {
  const trimmed = value.trim()
  if (trimmed === '') return null
  return /^-?\d+$/.test(trimmed) ? Number(trimmed) : 'invalid'
}

type RefuelFormProps = {
  refuel?: Refuel
  cars: Car[]
  initialCarId?: string
  stations: string[]
  submitLabel: string
  submittingLabel: string
  pending: boolean
  error: Error | null
  onSubmit: (input: RefuelInput) => void
  onEdit: () => void
  secondaryAction?: ReactNode
  // Rendered after the fields, such as the attachment queue of a create form.
  attachments?: ReactNode
}

export function RefuelForm({
  refuel,
  cars,
  initialCarId = '',
  stations,
  submitLabel,
  submittingLabel,
  pending,
  error,
  onSubmit,
  onEdit,
  secondaryAction,
  attachments,
}: RefuelFormProps) {
  const { t } = useTranslation()
  const units = useUnitFormat()
  const [values, setValues] = useState(() =>
    initialValues(refuel, initialCarId),
  )
  const [localReasons, setLocalReasons] = useState<Record<string, string>>({})

  const serverReasons =
    error instanceof ApiError && error.status === 400
      ? (error.fields ?? {})
      : {}
  const reasons = { ...serverReasons, ...localReasons }
  const invalidFields = fieldNames.filter((name) => reasons[name])
  const formError = error !== null && invalidFields.length === 0

  function change(name: FieldName, value: string) {
    setValues((current) => ({ ...current, [name]: value }))
    setLocalReasons(({ [name]: _, ...rest }) => rest)
    onEdit()
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const odometerReading = odometer(values.odometerReading)
    if (odometerReading === 'invalid') {
      setLocalReasons({ odometerReading: 'invalid_type' })
      return
    }
    onSubmit({
      carId: values.carId,
      date: fromLocalInput(values.date),
      station: values.station,
      odometerReading,
      fuel: values.fuel,
      liters: decimal(values.liters),
      amount: decimal(values.amount),
    })
  }

  function field(
    name: FieldName,
    control: (common: {
      id: string
      name: string
      'aria-invalid': boolean
      'aria-describedby': string | undefined
    }) => ReactNode,
    options: {
      optional?: boolean
      hint?: boolean
      label?: string
      suffix?: string
    } = {},
  ) {
    const id = `refuel-${name}`
    const invalid = invalidFields.includes(name)
    const suffixId = options.suffix ? `${id}-unit` : undefined
    const hintId = options.hint ? `${id}-hint` : undefined
    const errorId = invalid ? `${id}-error` : undefined
    const describedBy = [suffixId, hintId, errorId].filter(Boolean).join(' ')
    const reason = reasons[name]
    return (
      <div className="form-field">
        <label htmlFor={id}>
          {options.label ??
            t(`refuels.fields.${name === 'carId' ? 'car' : name}`)}
          {options.optional ? (
            <span className="field-optional"> ({t('refuels.optional')})</span>
          ) : null}
        </label>
        <div className="input-wrap">
          {control({
            id,
            name,
            'aria-invalid': invalid,
            'aria-describedby': describedBy || undefined,
          })}
          {suffixId && options.suffix ? (
            <InputSuffix id={suffixId}>{options.suffix}</InputSuffix>
          ) : null}
        </div>
        {hintId ? (
          <p className="field-hint" id={hintId}>
            {t(`refuels.hints.${name}`)}
          </p>
        ) : null}
        {errorId ? (
          <p className="field-error" id={errorId} role="alert">
            {t(
              `refuels.fieldErrors.${knownReasons.has(reason) ? reason : 'invalid'}`,
            )}
          </p>
        ) : null}
      </div>
    )
  }

  return (
    <form className="form-card" noValidate onSubmit={submit}>
      <div className="form-grid">
        {field('carId', (common) => (
          <select
            {...common}
            disabled={refuel !== undefined}
            required
            value={values.carId}
            onChange={(event) => change('carId', event.target.value)}
          >
            {values.carId === '' ? (
              <option value="">{t('refuels.chooseCar')}</option>
            ) : null}
            {cars.map((car) => (
              <option key={car.id} value={car.id}>
                {car.make} {car.name}
              </option>
            ))}
          </select>
        ))}
        {field('date', (common) => (
          <input
            {...common}
            required
            type="datetime-local"
            value={values.date}
            onChange={(event) => change('date', event.target.value)}
          />
        ))}
        {field(
          'station',
          (common) => (
            <>
              <input
                {...common}
                autoComplete="off"
                list="refuel-stations"
                required
                type="text"
                value={values.station}
                onChange={(event) => change('station', event.target.value)}
              />
              <datalist id="refuel-stations">
                {stations.map((station) => (
                  <option key={station} value={station} />
                ))}
              </datalist>
            </>
          ),
          { hint: true },
        )}
        {field(
          'odometerReading',
          (common) => (
            <input
              {...common}
              autoComplete="off"
              inputMode="numeric"
              type="text"
              value={values.odometerReading}
              onChange={(event) =>
                change('odometerReading', event.target.value)
              }
            />
          ),
          {
            optional: true,
            label: units.odometerReadingName,
            suffix: units.distanceSymbol,
          },
        )}
        {field('fuel', (common) => (
          <select
            {...common}
            value={values.fuel}
            onChange={(event) => change('fuel', event.target.value)}
          >
            {refuelFuels.map((fuel) => (
              <option key={fuel} value={fuel}>
                {t(`refuels.fuels.${fuel}`)}
              </option>
            ))}
          </select>
        ))}
        {field(
          'liters',
          (common) => (
            <input
              {...common}
              autoComplete="off"
              inputMode="decimal"
              required
              type="text"
              value={values.liters}
              onChange={(event) => change('liters', event.target.value)}
            />
          ),
          { hint: true, label: units.volumeName, suffix: units.volumeSymbol },
        )}
        {field(
          'amount',
          (common) => (
            <input
              {...common}
              autoComplete="off"
              inputMode="decimal"
              required
              type="text"
              value={values.amount}
              onChange={(event) => change('amount', event.target.value)}
            />
          ),
          { hint: true, suffix: units.currencySymbol },
        )}
      </div>

      {attachments}

      {formError ? (
        <p className="field-error" role="alert">
          {t('refuels.saveError')}
        </p>
      ) : null}

      <div className="form-actions">
        <button
          className="button button-primary"
          disabled={pending}
          type="submit"
        >
          {pending ? (
            <LoaderCircle className="spin" aria-hidden="true" size={18} />
          ) : (
            <Save aria-hidden="true" size={18} />
          )}
          {pending ? submittingLabel : submitLabel}
        </button>
        {secondaryAction}
      </div>
    </form>
  )
}
