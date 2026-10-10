## ADDED Requirements

### Requirement: Users have a role

The system SHALL store a `role` on every user, one of `user`, `moderator`, or `admin`,
defaulting to `user`. The role MUST be persisted in the database (not carried in the JWT)
so that a role change takes effect immediately on the next request. Roles are granted out
of band (direct database update); there is no self-service role-management API in this
change.

#### Scenario: New user defaults to the user role

- **WHEN** a new account is created (password or OAuth sign-in)
- **THEN** its role is `user`

### Requirement: Role-based authorization

The system SHALL provide a `RequireRole` authorization layer that runs after
authentication, reads the authenticated user id, loads the current role from the database,
and rejects the request when the role does not match the required one. Authentication
failure MUST surface as `401`; an authenticated user lacking the required role MUST surface
as `403`.

#### Scenario: Authorized role passes

- **WHEN** a request authenticated as a `moderator` hits an endpoint guarded by `RequireRole("moderator")`
- **THEN** the request proceeds to the handler

#### Scenario: Wrong role is forbidden

- **WHEN** a request authenticated as a `user` hits an endpoint guarded by `RequireRole("moderator")`
- **THEN** the system responds `403`

#### Scenario: Unauthenticated request is unauthorized

- **WHEN** a request with no valid credential hits an endpoint guarded by `RequireRole`
- **THEN** the system responds `401`
