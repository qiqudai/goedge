# UI E2E (Global Config)

This folder contains Playwright tests that verify global config pages auto-save on change
and persist after refresh. Tests modify values and restore them afterward.

## Setup

```
npm install
npx playwright install chromium
```

## Run

```
npm test
```

## Env

- `ADMIN_BASE_URL` (default: http://localhost:5173)
- `ADMIN_USER` (default: admin)
- `ADMIN_PASS` (default: 123456)
