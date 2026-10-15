import { LoaderCircle, Save } from 'lucide-react'
import { type FormEvent, type ReactNode, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  ApiError,
  type Car,
  type Ticket,
  type TicketInput,
  type TicketType,
  ticketTypes,
} from '#/api/client'
import { fromLocalInput, toLocalInput } from '#/repairs/format'

type Values = {
  carId: string
  date: string
  type: TicketType
  location: string
  amount: string
  description: string
}

type FieldName = keyof Values

const fieldNames: FieldName[] = [
  'carId',
  'date',
  'type',
  'location',
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

function initialValues(ticket: Ticket | undefined, carId: string): Values {
  return {
    carId: ticket?.carId ?? carId,
    date: toLocalInput(ticket ? new Date(ticket.date) : new Date()),
    type: ticket?.type ?? ticketTypes[0],
    location: ticket?.location ?? '',
    amount: ticket?.amount ?? '',
    description: ticket?.description ?? '',
  }
}

// Accept a decimal comma ("120,50"), as German users write amounts.
function decimal(value: string): string {
  const trimmed = value.trim()
  return /^-?\d+,\d+$/.test(trimmed) ? trimmed.replace(',', '.') : trimmed
}

type TicketFormProps = {
  ticket?: Ticket
  cars: Car[]
  initialCarId?: string
  locations: string[]
  submitLabel: string
  submittingLabel: string
  pending: boolean
  error: Error | null
  onSubmit: (input: TicketInput) => void
  onEdit: () => void
  secondaryAction?: ReactNode
  // Rendered after the fields, such as the attachment queue of a create form.
  attachments?: ReactNode
}

export function TicketForm({
  ticket,
  cars,
  initialCarId = '',
  locations,
  submitLabel,
  submittingLabel,
  pending,
  error,
  onSubmit,
  onEdit,
  secondaryAction,
  attachments,
}: TicketFormProps) {
  const { t } = useTranslation()
  const [values, setValues] = useState(() =>
    initialValues(ticket, initialCarId),
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
    onSubmit({
      carId: values.carId,
      date: fromLocalInput(values.date),
      type: values.type,
      location: values.location,
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
    options: { optional?: boolean; hint?: boolean; wide?: boolean } = {},
  ) {
    const id = `ticket-${name}`
    const invalid = invalidFields.includes(name)
    const hintId = options.hint ? `${id}-hint` : undefined
    const errorId = invalid ? `${id}-error` : undefined
    const describedBy = [hintId, errorId].filter(Boolean).join(' ')
    const reason = reasons[name]
    return (
      <div
        className={options.wide ? 'form-field form-field-wide' : 'form-field'}
      >
        <label htmlFor={id}>
          {t(`tickets.fields.${name === 'carId' ? 'car' : name}`)}
          {options.optional ? (
            <span className="field-optional"> ({t('tickets.optional')})</span>
          ) : null}
        </label>
        <div className="input-wrap">
          {control({
            id,
            name,
            'aria-invalid': invalid,
            'aria-describedby': describedBy || undefined,
          })}
        </div>
        {hintId ? (
          <p className="field-hint" id={hintId}>
            {t(`tickets.hints.${name}`)}
          </p>
        ) : null}
        {errorId ? (
          <p className="field-error" id={errorId} role="alert">
            {t(
              `tickets.fieldErrors.${knownReasons.has(reason) ? reason : 'invalid'}`,
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
            disabled={ticket !== undefined}
            required
            value={values.carId}
            onChange={(event) => change('carId', event.target.value)}
          >
            {values.carId === '' ? (
              <option value="">{t('tickets.chooseCar')}</option>
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
        {field('type', (common) => (
          <select
            {...common}
            value={values.type}
            onChange={(event) => change('type', event.target.value)}
          >
            {ticketTypes.map((type) => (
              <option key={type} value={type}>
                {t(`tickets.types.${type}`)}
              </option>
            ))}
          </select>
        ))}
        {field(
          'location',
          (common) => (
            <>
              <input
                {...common}
                autoComplete="off"
                list="ticket-locations"
                required
                type="text"
                value={values.location}
                onChange={(event) => change('location', event.target.value)}
              />
              <datalist id="ticket-locations">
                {locations.map((location) => (
                  <option key={location} value={location} />
                ))}
              </datalist>
            </>
          ),
          { hint: true },
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
          { hint: true },
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
          {t('tickets.saveError')}
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
