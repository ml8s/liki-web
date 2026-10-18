# Security Policy

## Reporting

Do not report security vulnerabilities through public GitHub issues.

Use GitHub private security reporting or the private contact channel configured in the repository settings.

Include:

- affected service or endpoint;
- reproduction steps;
- impact;
- a proof of concept without customer data;
- suggested mitigation if known.

## Scope

In scope:

- authentication or session handling issues;
- cross-user access;
- payment webhook forgery or replay;
- entitlement bypass;
- exposure of birth profile data;
- secret leakage;
- public MCP abuse that affects production.

Out of scope:

- vulnerability scanning against production;
- social engineering;
- distributed denial-of-service;
- issues requiring a compromised server or user device.

## Secrets

Never commit:

- JWT secrets;
- payment provider credentials;
- webhook signing keys;
- internal agent tokens;
- SQLite customer databases;
- production environment files.

## Security architecture

Architectural security decisions — header ownership, same-origin policy,
admin capability model, token and secret handling — are documented in
[ARCHITECTURE.md](./ARCHITECTURE.md). This file covers responsible disclosure
only.
