# user-auth (delta)

## ADDED Requirements

### Requirement: OAuth provider sign-in

The system SHALL support sign-in via external OAuth providers (Google, GitHub,
LinkedIn) using the server-side authorization-code flow, issuing the same
httpOnly cookie session as password auth.

- Each provider SHALL be enabled only when its client id and client secret are
  configured; routes for unknown or disabled providers respond `404`.
- The flow SHALL be protected against CSRF by a random `state` value carried in
  a short-lived httpOnly cookie and verified on callback.
- On success the callback SHALL set the session cookie and redirect the
  browser to the SPA; on any failure it SHALL redirect to the SPA with an
  `auth_error` query parameter, never rendering a JSON error to the browser.

#### Scenario: Start redirects to the provider

- **WHEN** a client requests `GET /api/v1/auth/oauth/:provider/start` for an enabled provider
- **THEN** the system sets a state cookie and responds `302` to the provider's consent URL carrying the same state

#### Scenario: Successful callback signs the user in

- **WHEN** the provider redirects back to the callback with a valid code and a state matching the state cookie
- **THEN** the system resolves the account, sets the httpOnly session cookie, and redirects to the SPA, where `GET /me` returns the user

#### Scenario: State mismatch is rejected

- **WHEN** the callback's `state` does not match the state cookie (or the cookie is absent)
- **THEN** the system sets no session cookie and redirects to the SPA with `auth_error`

#### Scenario: Unknown or disabled provider

- **WHEN** a client requests the start or callback route for a provider that is not configured
- **THEN** the system responds `404`

### Requirement: External identity account resolution

The system SHALL key external identities by `(provider, provider_user_id)` in
a `user_identities` table referencing `users`, resolving each OAuth sign-in to
exactly one account.

- A sign-in matching an existing identity SHALL resolve to its linked user.
- A first-time identity whose **verified** provider email matches an existing
  account (case-insensitive) SHALL be linked to that account.
- A first-time identity with no matching account SHALL create a new
  passwordless user (`password_hash` NULL) plus the identity, in one
  transaction.
- An identity whose provider email is unverified or absent SHALL NOT be linked
  by email and SHALL NOT create an account keyed to that email; the sign-in
  fails.

#### Scenario: Returning OAuth user

- **WHEN** a user signs in via a provider identity that already exists
- **THEN** the system starts a session for the linked user without touching `users` or creating rows

#### Scenario: First OAuth sign-in linking an existing password account

- **WHEN** a user signs in via a new provider identity whose verified email matches an existing account
- **THEN** the system adds the identity linked to that account and starts a session for it, leaving the password intact

#### Scenario: First OAuth sign-in creating a new account

- **WHEN** a user signs in via a new provider identity whose verified email matches no account
- **THEN** the system creates a passwordless user and the identity in one transaction and starts a session

#### Scenario: Unverified provider email never links

- **WHEN** a provider returns an identity whose email is unverified or missing
- **THEN** the system links nothing, creates nothing, and redirects to the SPA with `auth_error`

### Requirement: Enabled-provider listing

The system SHALL expose `GET /api/v1/auth/oauth/providers` returning the names
of currently enabled providers, so the web client renders only usable sign-in
buttons.

#### Scenario: Listing enabled providers

- **WHEN** a client requests the provider list while only Google and GitHub are configured
- **THEN** the system responds `200` with `{"data": ["google", "github"]}` (order not significant), omitting LinkedIn

### Requirement: Web client provider sign-in

The SPA auth dialog SHALL offer "Continue with <Provider>" actions for each
enabled provider alongside the email/password form, and SHALL surface a failed
OAuth callback (the `auth_error` redirect) as a user-visible message.

#### Scenario: Provider buttons reflect configuration

- **WHEN** a signed-out user opens the auth dialog
- **THEN** the dialog shows one sign-in action per enabled provider (and none for disabled providers) above the email/password form

#### Scenario: Provider sign-in round trip

- **WHEN** the user activates a provider action and completes consent at the provider
- **THEN** the browser returns to the SPA signed in (top bar shows the user's email) with no token ever exposed to JavaScript

#### Scenario: Failed callback is surfaced

- **WHEN** the SPA loads with `auth_error` in the URL after a failed callback
- **THEN** it shows a sign-in failure message and removes the parameter from the URL
