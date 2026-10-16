import { Description, Field, Label, Switch } from '@headlessui/react'
import { LoaderCircle, Save } from 'lucide-react'
import { type FormEvent, type ReactNode, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ApiError,
  type Car,
  type CarFuel,
  type CarInput,
  carFuels,
} from '#/api/client'
import { InputSuffix } from '#/components/input-suffix'
import { useUnitFormat } from '#/lib/display-units'

type Values = {
  type: string
  make: string
  name: string
  fuel: CarFuel
  firstRegistration: string
  licensePlate: string
  fin: string
  purchaseDate: string
  purchasePrice: string
  isActive: boolean
  deregistrationDate: string
  information: string
}

type FieldName = keyof Values

// Keeps a typed text in step with the backend's limit on information.
const maxInformationLength = 2000

const fields: {
  name: FieldName
  kind: 'text' | 'date' | 'select' | 'decimal' | 'switch' | 'textarea'
  optional?: boolean
  hint?: boolean
  wide?: boolean
}[] = [
  { name: 'type', kind: 'text', hint: true },
  { name: 'make', kind: 'text' },
  { name: 'name', kind: 'text' },
  { name: 'fuel', kind: 'select' },
  { name: 'firstRegistration', kind: 'date', optional: true },
  { name: 'licensePlate', kind: 'text', optional: true },
  { name: 'fin', kind: 'text', optional: true },
  { name: 'purchaseDate', kind: 'date', optional: true },
  { name: 'purchasePrice', kind: 'decimal', optional: true, hint: true },
  { name: 'isActive', kind: 'switch', hint: true },
  { name: 'deregistrationDate', kind: 'date', optional: true, hint: true },
  { name: 'information', kind: 'textarea', optional: true, wide: true },
]

const knownReasons = new Set([
  'required',
  'empty',
  'invalid_enum',
  'invalid_date',
  'invalid_decimal',
  'negative',
  'invalid_type',
  'too_long',
  'before_first_registration',
  'fuel_locked',
])

function initialValues(car?: Car): Values {
  return {
    type: car?.type ?? '',
    make: car?.make ?? '',
    name: car?.name ?? '',
    fuel: car?.fuel ?? carFuels[0],
    firstRegistration: car?.firstRegistration ?? '',
    licensePlate: car?.licensePlate ?? '',
    fin: car?.fin ?? '',
    purchaseDate: car?.purchaseDate ?? '',
    purchasePrice: car?.purchasePrice ?? '',
    isActive: car?.isActive ?? true,
    deregistrationDate: car?.deregistrationDate ?? '',
    information: car?.information ?? '',
  }
}

// Mirrors the backend's check, so the user is told before submitting.
// Full-dates order lexically.
function dateOrderReasons(values: Values): Record<string, string> {
  const first = values.firstRegistration.trim()
  const deregistration = values.deregistrationDate.trim()
  return first !== '' && deregistration !== '' && deregistration < first
    ? { deregistrationDate: 'before_first_registration' }
    : {}
}

function optional(value: string): string | null {
  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

// Accept a decimal comma ("18500,50"), as German users write amounts.
function decimal(value: string): string {
  return /^-?\d+,\d+$/.test(value) ? value.replace(',', '.') : value
}

function toInput(values: Values): CarInput {
  const purchasePrice = optional(values.purchasePrice)
  return {
    type: values.type,
    make: values.make,
    name: values.name,
    fuel: values.fuel,
    firstRegistration: optional(values.firstRegistration),
    licensePlate: optional(values.licensePlate),
    fin: optional(values.fin),
    purchaseDate: optional(values.purchaseDate),
    purchasePrice: purchasePrice === null ? null : decimal(purchasePrice),
    isActive: values.isActive,
    deregistrationDate: optional(values.deregistrationDate),
    information: optional(values.information),
  }
}

type CarFormProps = {
  car?: Car
  submitLabel: string
  submittingLabel: string
  pending: boolean
  error: Error | null
  onSubmit: (input: CarInput) => void
  onEdit: () => void
  secondaryAction?: ReactNode
  // Rendered after the fields, such as the attachment queue of a create form.
  attachments?: ReactNode
}

export function CarForm({
  car,
  submitLabel,
  submittingLabel,
  pending,
  error,
  onSubmit,
  onEdit,
  secondaryAction,
  attachments,
}: CarFormProps) {
  const { t } = useTranslation()
  const units = useUnitFormat()
  const [values, setValues] = useState(() => initialValues(car))
  // A car with refuels cannot switch to or from electric: its refuels'
  // units and subtypes follow the fuel. The backend enforces this too.
  const refuelCount = car?.refuelCount ?? 0
  const fuelLocked = (fuel: CarFuel) =>
    refuelCount > 0 && (fuel === 'electric') !== (car?.fuel === 'electric')
  const [localReasons, setLocalReasons] = useState<Record<string, string>>({})

  const reasons = {
    ...(error instanceof ApiError && error.status === 400
      ? (error.fields ?? {})
      : {}),
    ...localReasons,
  }
  const invalidFields = fields
    .map((field) => field.name)
    .filter((name) => reasons[name])
  const formError =
    error !== null &&
    Object.keys(localReasons).length === 0 &&
    invalidFields.length === 0

  function change<Name extends FieldName>(name: Name, value: Values[Name]) {
    setValues((current) => ({ ...current, [name]: value }))
    setLocalReasons({})
    onEdit()
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const dateReasons = dateOrderReasons(values)
    setLocalReasons(dateReasons)
    if (Object.keys(dateReasons).length > 0) return
    onSubmit(toInput(values))
  }

  return (
    <form className="form-card" noValidate onSubmit={submit}>
      <div className="form-grid">
        {fields.map(({ name, kind, optional: isOptional, hint, wide }) => {
          if (name === 'isActive') {
            return (
              <Field className="form-field" key={name}>
                <Label>{t('cars.fields.isActive')}</Label>
                <div className="switch-row">
                  <Switch
                    className="switch"
                    checked={values.isActive}
                    onChange={(checked) => change('isActive', checked)}
                  >
                    <span className="switch-thumb" aria-hidden="true" />
                  </Switch>
                  <span aria-hidden="true">
                    {t(
                      values.isActive
                        ? 'cars.states.active'
                        : 'cars.states.inactive',
                    )}
                  </span>
                </div>
                <Description className="field-hint">
                  {t('cars.hints.isActive')}
                </Description>
              </Field>
            )
          }
          const id = `car-${name}`
          const invalid = invalidFields.includes(name)
          // Only the purchase price is a decimal field; it is an amount.
          const suffixId = kind === 'decimal' ? `${id}-unit` : undefined
          const hintId = hint ? `${id}-hint` : undefined
          const lockId =
            kind === 'select' && refuelCount > 0 ? `${id}-locked` : undefined
          const errorId = invalid ? `${id}-error` : undefined
          const describedBy = [suffixId, hintId, lockId, errorId]
            .filter(Boolean)
            .join(' ')
          const reason = reasons[name]
          const common = {
            id,
            name,
            'aria-invalid': invalid,
            'aria-describedby': describedBy || undefined,
          }
          return (
            <div
              className={wide ? 'form-field form-field-wide' : 'form-field'}
              key={name}
            >
              <label htmlFor={id}>
                {t(`cars.fields.${name}`)}
                {isOptional ? (
                  <span className="field-optional">
                    {' '}
                    ({t('cars.optional')})
                  </span>
                ) : null}
              </label>
              <div className="input-wrap">
                {kind === 'select' ? (
                  <select
                    {...common}
                    value={values.fuel}
                    onChange={(event) => {
                      const fuel = carFuels.find(
                        (option) => option === event.target.value,
                      )
                      if (fuel) change('fuel', fuel)
                    }}
                  >
                    {carFuels.map((fuel) => (
                      <option
                        key={fuel}
                        value={fuel}
                        disabled={fuelLocked(fuel)}
                      >
                        {t(`cars.fuels.${fuel}`)}
                      </option>
                    ))}
                  </select>
                ) : kind === 'textarea' ? (
                  <textarea
                    {...common}
                    maxLength={maxInformationLength}
                    value={values[name]}
                    onChange={(event) => change(name, event.target.value)}
                  />
                ) : (
                  <input
                    {...common}
                    autoComplete="off"
                    inputMode={kind === 'decimal' ? 'decimal' : undefined}
                    required={!isOptional}
                    type={kind === 'date' ? 'date' : 'text'}
                    value={values[name]}
                    onChange={(event) => change(name, event.target.value)}
                  />
                )}
                {suffixId ? (
                  <InputSuffix id={suffixId}>
                    {units.currencySymbol}
                  </InputSuffix>
                ) : null}
              </div>
              {hintId ? (
                <p className="field-hint" id={hintId}>
                  {t(`cars.hints.${name}`)}
                </p>
              ) : null}
              {lockId ? (
                <p className="field-hint" id={lockId}>
                  {t('cars.hints.fuelLocked', { count: refuelCount })}
                </p>
              ) : null}
              {errorId ? (
                <p className="field-error" id={errorId} role="alert">
                  {t(
                    `cars.fieldErrors.${knownReasons.has(reason) ? reason : 'invalid'}`,
                  )}
                </p>
              ) : null}
            </div>
          )
        })}
      </div>

      {attachments}

      {formError ? (
        <p className="field-error" role="alert">
          {t('cars.saveError')}
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
