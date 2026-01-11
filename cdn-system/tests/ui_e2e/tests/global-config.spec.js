const { test, expect } = require('@playwright/test')

const adminUser = process.env.ADMIN_USER || 'admin'
const adminPass = process.env.ADMIN_PASS || '123456'

function logStep(message) {
  console.log(`[ui-e2e] ${message}`)
}

function trackSaveResponses(page) {
  let count = 0
  const handler = (resp) => {
    const url = resp.url()
    const isSave = url.includes('/global_config') || url.includes('/config_items')
    if (isSave && resp.request().method() === 'POST') {
      count += 1
    }
  }
  page.on('response', handler)
  return {
    get count() {
      return count
    },
    stop() {
      page.off('response', handler)
    }
  }
}

async function waitForCount(locator, minCount, timeoutMs = 10000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    if (await locator.count() >= minCount) {
      return
    }
    await new Promise((r) => setTimeout(r, 200))
  }
  throw new Error(`Expected at least ${minCount} elements, got ${await locator.count()}`)
}

async function login(page) {
  await page.goto('/login')
  await page.fill('input[placeholder="Username"]', adminUser)
  await page.fill('input[placeholder="Password"]', adminPass)
  await Promise.all([
    page.waitForURL('**/dashboard', { timeout: 15000 }),
    page.click('button:has-text("Login")')
  ])
}

async function waitForSave(page, timeoutMs = 5000, strict = false) {
  const result = await page
    .waitForResponse(
      (resp) => {
        const url = resp.url()
        const isSave = url.includes('/global_config') || url.includes('/config_items')
        return isSave && resp.request().method() === 'POST'
      },
      { timeout: timeoutMs }
    )
    .catch(() => null)
  if (!result && strict) {
    throw new Error('Save request not observed')
  }
  await page.waitForTimeout(300)
}

async function collectTextInputs(page) {
  const locator = page.locator('input.el-input__inner')
  const items = []
  const count = await locator.count()
  for (let i = 0; i < count; i += 1) {
    const input = locator.nth(i)
    const info = await input.evaluate((el) => {
      const isReadonly = el.hasAttribute('readonly')
      const isDisabled = el.hasAttribute('disabled')
      const isNumber = !!el.closest('.el-input-number')
      const isSelect = !!el.closest('.el-select')
      return { isReadonly, isDisabled, isNumber, isSelect }
    })
    if (info.isReadonly || info.isDisabled || info.isNumber || info.isSelect) {
      continue
    }
    const value = await input.evaluate((el) => el.value)
    items.push({ input, value })
  }
  return items
}

async function collectNumberInputs(page) {
  const locator = page.locator('.el-input-number input.el-input__inner')
  const items = []
  const count = await locator.count()
  for (let i = 0; i < count; i += 1) {
    const input = locator.nth(i)
    if (!(await input.isVisible())) {
      continue
    }
    const info = await input.evaluate((el) => {
      const isReadonly = el.hasAttribute('readonly')
      const isDisabled = el.hasAttribute('disabled')
      return { isReadonly, isDisabled }
    })
    if (info.isReadonly || info.isDisabled) {
      continue
    }
    const raw = await input.evaluate((el) => el.value)
    items.push({ input, value: raw })
  }
  return items
}

async function collectTextareas(page) {
  const locator = page.locator('textarea')
  const items = []
  const count = await locator.count()
  for (let i = 0; i < count; i += 1) {
    const input = locator.nth(i)
    const info = await input.evaluate((el) => {
      const isReadonly = el.hasAttribute('readonly')
      const isDisabled = el.hasAttribute('disabled')
      return { isReadonly, isDisabled }
    })
    if (info.isReadonly || info.isDisabled) {
      continue
    }
    const value = await input.evaluate((el) => el.value)
    items.push({ input, value })
  }
  return items
}

async function collectSwitches(page) {
  const locator = page.locator('.el-switch')
  const items = []
  const count = await locator.count()
  for (let i = 0; i < count; i += 1) {
    const sw = locator.nth(i)
    if (!(await sw.isVisible())) {
      continue
    }
    const info = await sw.evaluate((el) => ({
      disabled: el.classList.contains('is-disabled'),
      checked: el.classList.contains('is-checked')
    }))
    if (info.disabled) continue
    items.push({ sw, checked: info.checked })
  }
  return items
}

async function collectRadioGroups(page) {
  const locator = page.locator('.el-radio-group')
  const groups = []
  const count = await locator.count()
  for (let i = 0; i < count; i += 1) {
    const group = locator.nth(i)
    groups.push(group)
  }
  return groups
}

async function collectSelects(page) {
  const locator = page.locator('.el-select')
  const items = []
  const count = await locator.count()
  for (let i = 0; i < count; i += 1) {
    const select = locator.nth(i)
    const info = await select.evaluate((el) => ({
      disabled: el.classList.contains('is-disabled')
    }))
    if (info.disabled) continue
    const input = select.locator('input.el-input__inner')
    if ((await input.count()) === 0) {
      continue
    }
    const value = await input.first().evaluate((el) => el.value)
    items.push({ select, value })
  }
  return items
}

async function toggleSwitches(page, waitEach = true) {
  const switches = await collectSwitches(page)
  for (const item of switches) {
    await item.sw.click()
    if (waitEach) {
      await waitForSave(page)
    }
  }
  if (!waitEach && switches.length > 0) {
    await waitForSave(page)
  }
  return switches
}

async function restoreSwitches(page, switches) {
  const current = await collectSwitches(page)
  for (let i = 0; i < current.length && i < switches.length; i += 1) {
    const before = switches[i].checked
    const now = await current[i].sw.evaluate((el) => el.classList.contains('is-checked'))
    if (before !== now) {
      await current[i].sw.click()
      await waitForSave(page)
    }
  }
}

async function updateTextInputs(page, suffix, waitEach = true) {
  const inputs = await collectTextInputs(page)
  const updated = []
  for (const item of inputs) {
    const next = `${item.value}${suffix}`
    await item.input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, next)
    if (waitEach) {
      await waitForSave(page)
    }
    updated.push({ value: item.value, expected: next })
  }
  if (!waitEach && updated.length > 0) {
    await waitForSave(page)
  }
  return updated
}

async function updateTextareas(page, suffix, waitEach = true) {
  const inputs = await collectTextareas(page)
  const updated = []
  for (const item of inputs) {
    const next = `${item.value}${suffix}`
    await item.input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, next)
    if (waitEach) {
      await waitForSave(page)
    }
    updated.push({ value: item.value, expected: next })
  }
  if (!waitEach && updated.length > 0) {
    await waitForSave(page)
  }
  return updated
}

async function updateNumberInputs(page, waitEach = true) {
  const inputs = await collectNumberInputs(page)
  const updated = []
  for (const item of inputs) {
    const raw = String(item.value || '').trim()
    const num = Number.parseInt(raw || '0', 10)
    const next = Number.isNaN(num) ? '1' : String(num + 1)
    await item.input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, next)
    if (waitEach) {
      await waitForSave(page)
    }
    updated.push({ value: item.value, expected: next })
  }
  if (!waitEach && updated.length > 0) {
    await waitForSave(page)
  }
  return updated
}

async function updateSelects(page, waitEach = true) {
  const selects = await collectSelects(page)
  const updated = []
  for (const item of selects) {
    await item.select.click()
    const option = page.locator('.el-select-dropdown__item:not(.selected):not(.is-disabled)').first()
    if (await option.count()) {
      const text = await option.textContent()
      await option.click()
      if (waitEach) {
        await waitForSave(page)
      }
      updated.push({ original: item.value, expected: text?.trim() || '' })
    } else {
      updated.push({ original: item.value, expected: item.value })
    }
  }
  if (!waitEach && updated.length > 0) {
    await waitForSave(page)
  }
  return updated
}

async function updateRadioGroups(page, waitEach = true) {
  const groups = await collectRadioGroups(page)
  const updated = []
  for (const group of groups) {
    if (!(await group.isVisible())) {
      continue
    }
    const checked = group.locator('.el-radio.is-checked')
    const next = group.locator('.el-radio:not(.is-checked)')
    const checkedCount = await checked.count()
    const original = checkedCount ? await checked.first().textContent() : ''
    if (await next.count()) {
      if (await next.first().isVisible()) {
        await next.first().click()
      } else {
        continue
      }
      if (waitEach) {
        await waitForSave(page)
      }
      const updatedChecked = group.locator('.el-radio.is-checked')
      const nextLabel = await updatedChecked.first().textContent()
      updated.push({
        original: (original || '').trim(),
        expected: (nextLabel || '').trim()
      })
    } else if (await checked.count()) {
      const label = await checked.first().textContent()
      updated.push({ original: (label || '').trim(), expected: (label || '').trim() })
    }
  }
  if (!waitEach && updated.length > 0) {
    await waitForSave(page)
  }
  return updated
}

async function expectTextInputs(page, expected) {
  const inputs = await collectTextInputs(page)
  expect(inputs.length).toBe(expected.length)
  for (let i = 0; i < inputs.length; i += 1) {
    const val = await inputs[i].input.evaluate((el) => el.value)
    expect(val).toBe(expected[i].expected)
  }
}

async function expectTextareas(page, expected) {
  const inputs = await collectTextareas(page)
  expect(inputs.length).toBe(expected.length)
  for (let i = 0; i < inputs.length; i += 1) {
    const val = await inputs[i].input.evaluate((el) => el.value)
    expect(val).toBe(expected[i].expected)
  }
}

async function expectNumberInputs(page, expected) {
  const inputs = await collectNumberInputs(page)
  expect(inputs.length).toBe(expected.length)
  for (let i = 0; i < inputs.length; i += 1) {
    const val = await inputs[i].input.evaluate((el) => el.value)
    expect(val).toBe(expected[i].expected)
  }
}

async function expectSelects(page, expected) {
  const selects = await collectSelects(page)
  expect(selects.length).toBe(expected.length)
  for (let i = 0; i < selects.length; i += 1) {
    const val = await selects[i].select.locator('input.el-input__inner').evaluate((el) => el.value)
    expect(val).toBe(expected[i].expected)
  }
}

async function expectSwitchesToggled(page, originals) {
  const current = await collectSwitches(page)
  expect(current.length).toBe(originals.length)
  for (let i = 0; i < current.length; i += 1) {
    expect(current[i].checked).toBe(!originals[i].checked)
  }
}

async function expectRadioGroups(page, expected) {
  const groups = await collectRadioGroups(page)
  const visibleGroups = []
  for (const group of groups) {
    if (await group.isVisible()) {
      visibleGroups.push(group)
    }
  }
  expect(visibleGroups.length).toBe(expected.length)
  for (let i = 0; i < visibleGroups.length; i += 1) {
    const checked = visibleGroups[i].locator('.el-radio.is-checked')
    const checkedCount = await checked.count()
    const label = checkedCount ? await checked.first().textContent() : ''
    expect((label || '').trim()).toBe(expected[i].expected)
  }
}

async function restoreSelects(page, selects) {
  const current = await collectSelects(page)
  for (let i = 0; i < current.length && i < selects.length; i += 1) {
    const original = selects[i].original
    if (!original) continue
    await current[i].select.click()
    const target = page.locator('.el-select-dropdown__item:not(.is-disabled)').filter({ hasText: original }).first()
    if (await target.count()) {
      await target.click()
      await waitForSave(page)
    } else {
      await page.keyboard.press('Escape')
    }
  }
}

async function restoreRadioGroups(page, radios) {
  const groups = await collectRadioGroups(page)
  const visibleGroups = []
  for (const group of groups) {
    if (await group.isVisible()) {
      visibleGroups.push(group)
    }
  }
  for (let i = 0; i < visibleGroups.length && i < radios.length; i += 1) {
    const original = radios[i].original
    if (!original) continue
    const target = visibleGroups[i].locator('.el-radio').filter({ hasText: original }).first()
    if ((await target.count()) && (await target.isVisible())) {
      await target.click()
      await waitForSave(page)
    }
  }
}

test.beforeEach(async ({ page }) => {
  await login(page)
})

test('global errors autosave', async ({ page }) => {
  logStep('errors: open')
  await page.goto('/global/errors')
  await waitForCount(page.locator('textarea'), 1)
  logStep('errors: update textareas')
  const updated = await updateTextareas(page, '\n<!-- e2e -->', false)
  logStep('errors: reload')
  await page.reload()
  await waitForCount(page.locator('textarea'), 1)
  logStep('errors: verify')
  await expectTextareas(page, updated)
  const restore = await collectTextareas(page)
  logStep('errors: restore')
  for (let i = 0; i < restore.length && i < updated.length; i += 1) {
    await restore[i].input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, updated[i].value)
  }
  if (restore.length > 0) {
    await waitForSave(page)
  }
  logStep('errors: done')
})

test('global resources autosave', async ({ page }) => {
  logStep('resources: open')
  await page.goto('/global/resources')
  await waitForCount(page.locator('.el-form'), 1)
  logStep('resources: update inputs')
  const textUpdated = await updateTextInputs(page, '-e2e', false)
  const numUpdated = await updateNumberInputs(page, false)
  const switchState = await toggleSwitches(page, false)
  logStep('resources: reload')
  await page.reload()
  await waitForCount(page.locator('.el-form'), 1)
  logStep('resources: verify')
  await expectTextInputs(page, textUpdated)
  await expectNumberInputs(page, numUpdated)
  logStep('resources: restore switches')
  await restoreSwitches(page, switchState)

  const restoreText = await collectTextInputs(page)
  logStep('resources: restore inputs')
  for (let i = 0; i < restoreText.length && i < textUpdated.length; i += 1) {
    await restoreText[i].input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, textUpdated[i].value)
  }
  if (restoreText.length > 0) {
    await waitForSave(page)
  }

  const restoreNum = await collectNumberInputs(page)
  for (let i = 0; i < restoreNum.length && i < numUpdated.length; i += 1) {
    await restoreNum[i].input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, String(numUpdated[i].value))
  }
  if (restoreNum.length > 0) {
    await waitForSave(page)
  }
  logStep('resources: done')
})

test('global firewall autosave', async ({ page }) => {
  logStep('firewall: open')
  await page.goto('/global/firewall')
  await waitForCount(page.locator('.el-form'), 1)
  logStep('firewall: update inputs')
  const saveTracker = trackSaveResponses(page)
  const textUpdated = await updateTextInputs(page, '-e2e', false)
  const textareasUpdated = await updateTextareas(page, '\n#e2e', false)
  const numUpdated = await updateNumberInputs(page, false)
  const selectUpdated = await updateSelects(page, false)
  const radioUpdated = await updateRadioGroups(page, true)
  await page.waitForTimeout(1500)
  const saveCount = saveTracker.count
  saveTracker.stop()
  if (saveCount === 0) {
    throw new Error('No save requests observed for firewall updates')
  }

  logStep('firewall: reload')
  await page.reload()
  await waitForCount(page.locator('.el-form'), 1)
  logStep('firewall: verify')
  await expectTextInputs(page, textUpdated)
  await expectTextareas(page, textareasUpdated)
  await expectNumberInputs(page, numUpdated)
  await expectSelects(page, selectUpdated)
  await expectRadioGroups(page, radioUpdated)

  const restoreText = await collectTextInputs(page)
  for (let i = 0; i < restoreText.length && i < textUpdated.length; i += 1) {
    await restoreText[i].input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, textUpdated[i].value)
  }
  if (restoreText.length > 0) {
    await waitForSave(page)
  }

  const restoreTextareas = await collectTextareas(page)
  for (let i = 0; i < restoreTextareas.length && i < textareasUpdated.length; i += 1) {
    await restoreTextareas[i].input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, textareasUpdated[i].value)
  }
  if (restoreTextareas.length > 0) {
    await waitForSave(page)
  }

  const restoreNum = await collectNumberInputs(page)
  for (let i = 0; i < restoreNum.length && i < numUpdated.length; i += 1) {
    await restoreNum[i].input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, String(numUpdated[i].value))
  }
  if (restoreNum.length > 0) {
    await waitForSave(page)
  }

  logStep('firewall: verify switches')
  const switchState = await toggleSwitches(page, false)
  await page.reload()
  await waitForCount(page.locator('.el-form'), 1)
  await expectSwitchesToggled(page, switchState)
  await restoreSwitches(page, switchState)
  logStep('firewall: restore selects/radios')
  await restoreSelects(page, selectUpdated)
  await restoreRadioGroups(page, radioUpdated)
  logStep('firewall: done')
})

test('global nginx autosave', async ({ page }) => {
  logStep('nginx: open')
  await page.goto('/global/nginx')
  await waitForCount(page.locator('.el-form'), 1)
  logStep('nginx: update inputs')
  const textUpdated = await updateTextInputs(page, '-e2e', false)
  const numUpdated = await updateNumberInputs(page, false)
  const switchState = await toggleSwitches(page, false)
  logStep('nginx: reload')
  await page.reload()
  logStep('nginx: verify')
  await expectTextInputs(page, textUpdated)
  await expectNumberInputs(page, numUpdated)
  logStep('nginx: restore switches')
  await restoreSwitches(page, switchState)

  const restoreText = await collectTextInputs(page)
  for (let i = 0; i < restoreText.length && i < textUpdated.length; i += 1) {
    await restoreText[i].input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, textUpdated[i].value)
  }
  if (restoreText.length > 0) {
    await waitForSave(page)
  }

  const restoreNum = await collectNumberInputs(page)
  for (let i = 0; i < restoreNum.length && i < numUpdated.length; i += 1) {
    await restoreNum[i].input.evaluate((el, value) => {
      el.focus()
      el.value = value
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      el.dispatchEvent(new Event('blur'))
      el.blur()
    }, String(numUpdated[i].value))
  }
  if (restoreNum.length > 0) {
    await waitForSave(page)
  }
  logStep('nginx: done')
})
