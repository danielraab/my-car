# api-contract/profile Specification

## Purpose

Defines the authenticated caller's profile representation and safe update operation used by the rewrite's profile screen.

## Requirements

### Requirement: Caller can retrieve the current profile
The system SHALL expose `GET /api/v1/me` for the authenticated account and
return its ID, email, first name, last name, currency, volume unit, and
distance unit without exposing session or identity-provider credentials. The
currency SHALL be an uppercase ISO 4217 alphabetic code; the volume unit SHALL
be `liter` or `gallon`; the distance unit SHALL be `kilometer` or `mile`. The
session representation's embedded profile SHALL contain the same members.
The OpenAPI descriptions of refuel `liters`, `perLiter` and `consumption`, and
of every `odometerReading` and `distance`, SHALL state that the values are in
the account's configured currency and units and are never converted.

#### Scenario: Authenticated caller reads profile
- **WHEN** an authenticated caller requests `GET /api/v1/me`
- **THEN** the response is `200` with that caller's profile, including
  `currency`, `volumeUnit` and `distanceUnit`

#### Scenario: Account that never chose settings
- **WHEN** an account that has never changed its display settings requests
  `GET /api/v1/me`
- **THEN** the profile contains `currency` `EUR`, `volumeUnit` `liter` and
  `distanceUnit` `kilometer`

### Requirement: Caller can update the current profile
The system SHALL expose `PATCH /api/v1/me` for first name, last name,
currency, volume unit, and distance unit updates only. The email address
SHALL NOT be changeable through this operation: a request body that contains
an `email` member SHALL be rejected with `400` and the common validation error
representation identifying the `email` field, and no part of the request
SHALL be applied. A request body that contains any other undocumented member,
no documented member, or a non-string value for a documented member SHALL
likewise be rejected with `400` identifying the offending fields. Supplied
names SHALL have leading and trailing whitespace removed, SHALL be accepted
when empty rather than silently ignored, and SHALL be rejected with `400` when
longer than 100 characters after trimming. A supplied `currency` SHALL be
accepted only when it is an uppercase ISO 4217 alphabetic code known to the
backend; a supplied `volumeUnit` or `distanceUnit` SHALL be accepted only when
it is one of its documented values; any other value SHALL be rejected with
`400` identifying that field, and no part of the request SHALL be applied.
Changing a setting SHALL NOT change any stored car or expense value. A
successful update SHALL apply every supplied field and return the complete
updated profile.

#### Scenario: Caller updates their name
- **WHEN** an authenticated caller patches a valid first or last name
- **THEN** the response is `200` and contains the updated profile

#### Scenario: Caller clears an optional name
- **WHEN** an authenticated caller patches `lastName` to an empty string
- **THEN** the response is `200` and the profile's `lastName` is empty

#### Scenario: Caller changes display settings
- **WHEN** an authenticated caller patches `currency` to `USD`,
  `volumeUnit` to `gallon` and `distanceUnit` to `mile`
- **THEN** the response is `200` with those values, and the caller's refuels,
  repairs, tickets and cars return the same amounts, volumes and odometer
  readings as before

#### Scenario: Caller supplies an unknown currency
- **WHEN** an authenticated caller patches `currency` to `XYZ`, `usd`, or
  `EURO`
- **THEN** the response is `400` with the common error representation whose
  field details identify `currency`, and the stored profile is unchanged

#### Scenario: Caller supplies an unknown unit
- **WHEN** an authenticated caller patches `volumeUnit` to `barrel` or
  `distanceUnit` to `furlong`
- **THEN** the response is `400` identifying that field with `invalid_enum`
  and the stored profile is unchanged

#### Scenario: Caller attempts to change their email
- **WHEN** an authenticated caller patches a body containing `email`, with or
  without name fields
- **THEN** the response is `400` with the common error representation whose
  field details identify `email`, and the stored profile is unchanged

#### Scenario: Caller selects an occupied email
- **WHEN** an authenticated caller patches their email to one used by another
  account
- **THEN** the response is `400` (not `409`) with the common error
  representation whose field details identify `email` as read-only, without
  disclosing whether the address is in use

#### Scenario: Caller supplies an undocumented field
- **WHEN** an authenticated caller patches a body containing a member other
  than `firstName`, `lastName`, `currency`, `volumeUnit` or `distanceUnit`
- **THEN** the response is `400` identifying that member and the stored
  profile is unchanged

#### Scenario: Caller supplies an overlong name
- **WHEN** an authenticated caller patches a first or last name longer than
  100 characters after trimming
- **THEN** the response is `400` identifying that field and the stored
  profile is unchanged

#### Scenario: Unauthenticated caller updates a profile
- **WHEN** a request without a valid session patches `/api/v1/me`
- **THEN** the response is `401` with the common error representation
