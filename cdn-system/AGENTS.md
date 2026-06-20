# CDN System — Agent Guide

Before modifying delete guards, config precedence, site defaults, packages, CC/ACL, or node/line logic, read:

**[docs/BUSINESS_RULES.md](docs/BUSINESS_RULES.md)**

Cursor always-applies rule: `.cursor/rules/cdn-business-rules.mdc`

## Quick reference

- Config: `site settings > group default > user global default > platform template`
- Delete site: disable first
- Delete node: remove from all lines first
- Delete line: disable first
- Delete cert: disable + no site references
- Delete site group: no sites in group
- Delete CC/ACL: no references; ACL must be disabled
- Delete plan/user package: no downstream references

Implementation: `api/services/reference_guard_service.go`
