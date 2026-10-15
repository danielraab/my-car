import { LoaderCircle, Save } from 'lucide-react'
import { type FormEvent, type ReactNode, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ApiError,
  type Car,
  type Repair,
  type RepairInput,
  type RepairType,
  repairTypes,
} from '#/api/client'
import { InputSuffix } from '#/components/input-suffix'
import { useUnitFormat } from '#/lib/display-units'

import { fromLocalInput, toLocalInput } from './format'

type Values = {
  carId: string
  date: string
  station: string
  odometerReading: string
  type: RepairType
  amount: string
  description: string
}

type FieldName = keyof Values

const fieldNames: FieldName[] = [
  'carId',
  'date',
  'station',
  'odometerReading',
  'type',
  'amount',
  'description',
]

const knownReasons = new Set([
  'required',
  'empty',
  'invalid_enum',
  'invalid_decimal',
  'negative',
  'invalid_type',
])

function initialValues(repair: Repair | undefined, carId: string): Values {
  return {
    carId: repair?.carId ?? carId,
    date: toLocalInput(repair ? new Date(repair.date) : new Date()),
    station: repair?.station ?? '',
    odometerReading:
      repair?.odometerReading == null ? '' : String(repair.odometerReading),
    type: repair?.type ?? repairTypes[0],
    amount: repair?.amount ?? '',
    description: repair?.description ?? '',
  }
}

// Accept a decimal comma ("120,50"), as German users write amounts.
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

type RepairFormProps = {
  repair?: Repair
  cars: Car[]
  initialCarId?: string
  stations: string[]
  submitLabel: string
  submittingLabel: string
  pending: boolean
  error: Error | null
  onSubmit: (input: RepairInput) => void
  onEdit: () => void
  secondaryAction?: ReactNode
  // Rendered after the fields, such as the attachment queue of a create form.
  attachments?: ReactNode
}

export function RepairForm({
  repair,
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
}: RepairFormProps) {
  const { t } = useTranslation()
  const units = useUnitFormat()
  const [values, setValues] = useState(() =>
    initialValues(repair, initialCarId),
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
      type: values.type,
      amount: decimal(values.amount),
      description: values.description.trim(),
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
      wide?: boolean
      label?: string
      suffix?: string
    } = {},
  ) {
    const id = `repair-${name}`
    const invalid = invalidFields.includes(name)
    const suffixId = options.suffix ? `${id}-unit` : undefined
    const hintId = options.hint ? `${id}-hint` : undefined
    const errorId = invalid ? `${id}-error` : undefined
    const describedBy = [suffixId, hintId, errorId].filter(Boolean).join(' ')
    const reason = reasons[name]
    return (
      <div
        className={options.wide ? 'form-field form-field-wide' : 'form-field'}
      >
        <label htmlFor={id}>
          {options.label ??
            t(`repairs.fields.${name === 'carId' ? 'car' : name}`)}
          {options.optional ? (
            <span className="field-optional"> ({t('repairs.optional')})</span>
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
            {t(`repairs.hints.${name}`)}
          </p>
        ) : null}
        {errorId ? (
          <p className="field-error" id={errorId} role="alert">
            {t(
              `repairs.fieldErrors.${knownReasons.has(reason) ? reason : 'invalid'}`,
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
            disabled={repair !== undefined}
            required
            value={values.carId}
            onChange={(event) => change('carId', event.target.value)}
          >
            {values.carId === '' ? (
              <option value="">{t('repairs.chooseCar')}</option>
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
                list="repair-stations"
                required
                type="text"
                value={values.station}
                onChange={(event) => change('station', event.target.value)}
              />
              <datalist id="repair-stations">
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
        {field('type', (common) => (
          <select
            {...common}
            value={values.type}
            onChange={(event) => change('type', event.target.value)}
          >
            {repairTypes.map((type) => (
              <option key={type} value={type}>
                {t(`repairs.types.${type}`)}
              </option>
            ))}
          </select>
        ))}
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
        {field(
          'description',
          (common) => (
            <textarea
              {...common}
              value={values.description}
              onChange={(event) => change('description', event.target.value)}
            />
          ),
          { optional: true, wide: true },
        )}
      </div>

      {attachments}

      {formError ? (
        <p className="field-error" role="alert">
          {t('repairs.saveError')}
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
