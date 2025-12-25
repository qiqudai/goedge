---
name: cdnfly-admin-frontend-rule
description: Enforce CDNFly admin frontend rules and db.sql constraints when building or modifying web admin UI and CRUD flows in this project. Use for frontend pages in cdn-system/web/admin, especially user-related forms/lists, role-based visibility, and any feature that depends on the existing database schema (no schema changes).
---

# Cdnfly Admin Frontend Rule

## Overview

Apply strict frontend rules for CDNFly admin UI work. Always rely on the existing schema in `references/db.sql` and never add or expand tables/fields unless explicitly requested by the user. Use the admin-home demo as the UI reference: https://demo.cdnfly.cn/dashboard/admin-home

## Core Rules

1) Always determine the current role (admin vs user) before shaping UI behavior.
2) User role:
   - All CRUD actions target the current logged-in user only.
   - Never show the "user selection" control.
3) Admin role:
   - Show the "user selection" control.
   - Allow choosing/modifying the user via dropdown based on user info.
4) Package/plan selector:
   - Default select the first available plan; do not require the user to reselect.
5) Node sync requirement:
   - Any API changes for the following must trigger config sync to nodes:
     - Node management
     - Node group / line group management
     - Line management
     - Base package management
     - User package management
6) No direct node sync:
   - The following do not sync directly to nodes:
     - Region management
     - Admin user management
     - API key management

## Usage Checklist

- Confirm which role the page is operating under.
- Hide or show the user selector based on role (never show for user role).
- Bind CRUD operations to current user when role is user.
- Use `references/db.sql` as the only schema source unless the user explicitly asks for schema changes.
- Match UI layout and styling to the admin-home demo.
- Ensure APIs for node-related config changes trigger node sync logic.

## References

- `references/db.sql`: authoritative database schema for all frontend features.
