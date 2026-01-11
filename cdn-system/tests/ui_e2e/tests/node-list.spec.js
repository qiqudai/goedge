const { test, expect, request } = require('@playwright/test')

const adminUser = process.env.ADMIN_USER || 'admin'
const adminPass = process.env.ADMIN_PASS || '123456'
const apiBase = process.env.ADMIN_API_BASE || 'https://goai.665305.cc/api/v1/admin'

async function fetchAdminToken() {
  const req = await request.newContext()
  const resp = await req.post(`${apiBase}/login`, {
    data: { username: adminUser, password: adminPass }
  })
  const body = await resp.json()
  return body.token
}

async function setAdminStorage(page, token) {
  await page.addInitScript(([tokenValue]) => {
    localStorage.setItem('admin_token', tokenValue)
    localStorage.setItem('role', 'admin')
  }, [token])
}

async function waitForApi(page, urlFragment, method) {
  return page.waitForResponse((resp) => {
    if (!resp.url().includes(urlFragment)) return false
    if (method && resp.request().method() !== method) return false
    return true
  })
}

async function confirmDialog(page) {
  const dialog = page.getByRole('dialog', { name: '提示' })
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: '确定' }).click()
}

test('node list api actions work', async ({ page }) => {
  const token = await fetchAdminToken()
  await setAdminStorage(page, token)

  await page.goto('/node/list')
  await waitForApi(page, '/api/v1/admin/nodes', 'GET')

  const nodeName = `pw-node-${Date.now()}`
  const nodeNameUpdated = `${nodeName}-edit`

  await page.click('button:has-text("安装节点")')
  const nodeDialog = page.getByRole('dialog', { name: '创建新节点' })
  await expect(nodeDialog).toBeVisible()
  await nodeDialog.getByRole('textbox', { name: '名称' }).fill(nodeName)
  await nodeDialog.getByRole('textbox', { name: 'IP' }).fill('127.0.0.11')

  const createRespPromise = waitForApi(page, '/api/v1/admin/nodes', 'POST')
  await nodeDialog.getByRole('button', { name: '确认' }).click()
  const createResp = await createRespPromise
  const createData = await createResp.json()
  expect(createData.code).toBe(0)
  const nodeId = createData.data.id

  await expect(nodeDialog).toBeHidden()

  await page.fill('input[placeholder="节点名称 / IP"]', nodeName)
  const searchRespPromise = waitForApi(page, '/api/v1/admin/nodes', 'GET')
  await page.click('button:has-text("搜索")')
  await searchRespPromise

  const row = page.locator('.el-table__body-wrapper tbody tr').filter({ hasText: nodeName }).first()
  await expect(row).toBeVisible()
  await row.locator('.node-name-link').click()

  const editDialog = page.getByRole('dialog', { name: '编辑节点' })
  await expect(editDialog).toBeVisible()
  await editDialog.getByRole('textbox', { name: '名称' }).fill(nodeNameUpdated)

  const updateRespPromise = waitForApi(page, `/api/v1/admin/nodes/${nodeId}`, 'PUT')
  await editDialog.getByRole('button', { name: '确认' }).click()
  const updateResp = await updateRespPromise
  const updateData = await updateResp.json()
  expect(updateData.code).toBe(0)

  await expect(editDialog).toBeHidden()

  await page.fill('input[placeholder="节点名称 / IP"]', nodeNameUpdated)
  const refreshRespPromise = waitForApi(page, '/api/v1/admin/nodes', 'GET')
  await page.click('button:has-text("搜索")')
  await refreshRespPromise

  const updatedRow = page.locator('.el-table__body-wrapper tbody tr').filter({ hasText: nodeNameUpdated }).first()
  await expect(updatedRow).toBeVisible()

  const statusRespPromise = waitForApi(page, `/api/v1/admin/nodes/${nodeId}/status`, 'PUT')
  await updatedRow.locator('.el-switch').click()
  const statusResp = await statusRespPromise
  const statusData = await statusResp.json()
  expect(statusData.code).toBe(0)

  const monitorRespPromise = waitForApi(page, `/api/v1/admin/nodes/${nodeId}/monitor_logs`, 'GET')
  await updatedRow.locator('a:has-text("日志")').click()
  const monitorResp = await monitorRespPromise
  const monitorData = await monitorResp.json()
  expect(monitorData.code).toBe(0)
  const monitorDialog = page.getByRole('dialog', { name: '监控日志' })
  await monitorDialog.locator('.el-dialog__headerbtn').click()
  await expect(monitorDialog).toBeHidden()

  await updatedRow.locator('td .el-checkbox').first().click()
  await page.click('button:has-text("禁用节点")')
  await confirmDialog(page)
  const batchStopResp = await waitForApi(page, '/api/v1/admin/nodes/batch_action', 'POST')
  const batchStopData = await batchStopResp.json()
  expect(batchStopData.code).toBe(0)

  await updatedRow.locator('td .el-checkbox').first().click()
  const enableBtn = page.getByRole('button', { name: '启用节点' })
  await expect(enableBtn).toBeEnabled()
  await enableBtn.click()
  await confirmDialog(page)
  const batchStartResp = await waitForApi(page, '/api/v1/admin/nodes/batch_action', 'POST')
  const batchStartData = await batchStartResp.json()
  expect(batchStartData.code).toBe(0)

  await updatedRow.locator('.link-more').click()
  await page.locator('.el-dropdown-menu:visible .el-dropdown-menu__item').filter({ hasText: '删除' }).first().click()
  await confirmDialog(page)
  const deleteResp = await waitForApi(page, `/api/v1/admin/nodes/${nodeId}`, 'DELETE')
  const deleteData = await deleteResp.json()
  expect(deleteData.code).toBe(0)

  await page.click('.el-tabs__item:has-text("区域管理")')
  await page.waitForTimeout(1000)

  const regionName = `pw-region-${Date.now()}`
  await page.click('button:has-text("新增区域")')
  const regionDialog = page.getByRole('dialog', { name: '新增区域' })
  await expect(regionDialog).toBeVisible()
  await regionDialog.getByRole('textbox', { name: '名称' }).fill(regionName)
  await regionDialog.getByRole('textbox', { name: '备注' }).fill('pw')
  const regionCreateRespPromise = waitForApi(page, '/api/v1/admin/regions', 'POST')
  await regionDialog.locator('button:has-text("确定")').click()
  const regionCreateResp = await regionCreateRespPromise
  const regionCreateData = await regionCreateResp.json()
  expect(regionCreateData.code).toBe(0)

  const regionRow = page.locator('.el-table__body-wrapper tbody tr').filter({ hasText: regionName }).first()
  await expect(regionRow).toBeVisible()
  await regionRow.locator('button:has-text("删除")').click()
  await confirmDialog(page)
  const regionDeleteResp = await waitForApi(page, '/api/v1/admin/regions/', 'DELETE')
  const regionDeleteData = await regionDeleteResp.json()
  expect(regionDeleteData.code).toBe(0)
})
