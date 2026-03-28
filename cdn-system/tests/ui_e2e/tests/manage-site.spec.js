const { test, expect } = require('@playwright/test')
const { execFile } = require('node:child_process')
const fs = require('node:fs')

const adminUser = process.env.ADMIN_USER || 'admin'
const adminPass = process.env.ADMIN_PASS || '123456'
const siteId = process.env.SITE_ID || '37'
const domainHost = process.env.SITE_DOMAIN || 'testabc.665305.cc'
const originHost = process.env.ORIGIN_HOST || '202.73.4.80'
const edgePort = process.env.EDGE_PORT || '81'
const certPath = process.env.CERT_PATH || '/tmp/testabc.crt'
const keyPath = process.env.KEY_PATH || '/tmp/testabc.key'
const agentConfigPath = process.env.AGENT_CONFIG_PATH
  || '/www/server/go_project/openresty/cdn-system/agent/edge-node/conf/cdn_config.json'

const managePath = `/website/manage?site_id=${siteId}`

test.describe.configure({ mode: 'serial' })
test.use({ ignoreHTTPSErrors: true })

function logStep(message) {
  console.log(`[manage-e2e] ${message}`)
}

function wait(ms) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

async function selectRadioIfNeeded(page, labelLocator) {
  const wrapper = labelLocator.locator('xpath=ancestor::label[contains(@class,\"el-radio\")]')
  const isChecked = await wrapper.evaluate(el => el.classList.contains('is-checked'))
  if (!isChecked) {
    await waitForSave(page, () => labelLocator.click())
    return true
  }
  return false
}

async function ensureSwitch(page, switchLocator, expected) {
  const isChecked = await switchLocator.evaluate(el => el.classList.contains('is-checked'))
  if (isChecked !== expected) {
    await waitForSave(page, () => switchLocator.click())
    return true
  }
  return false
}

async function login(page) {
  await page.goto('/login')
  await page.fill('input[placeholder="Username"]', adminUser)
  await page.fill('input[placeholder="Password"]', adminPass)
  await Promise.all([
    page.waitForURL('**/dashboard', { timeout: 30000 }),
    page.click('button:has-text("Login")')
  ])
}

async function gotoManage(page) {
  await page.goto(managePath)
  await page.waitForSelector('.site-manage', { timeout: 30000 })
}

async function waitForSave(page, action) {
  const responsePromise = page.waitForResponse(
    resp => resp.url().includes(`/sites/${siteId}`) && resp.request().method() === 'PUT',
    { timeout: 30000 }
  )
  if (action) {
    await action()
  }
  await responsePromise
  await wait(300)
}

async function readAgentConfig() {
  const raw = fs.readFileSync(agentConfigPath, 'utf-8')
  return JSON.parse(raw)
}

async function waitForAgentVersion(prevVersion) {
  const start = Date.now()
  while (Date.now() - start < 30000) {
    try {
      const config = await readAgentConfig()
      if (config.version && config.version !== prevVersion) {
        return config
      }
    } catch (err) {
      // ignore transient read errors
    }
    await wait(500)
  }
  throw new Error('Timed out waiting for agent config update')
}

async function waitForDomainUpdate(predicate) {
  const start = Date.now()
  while (Date.now() - start < 30000) {
    try {
      const config = await readAgentConfig()
      const domain = await getDomainConfig(config)
      if (predicate(domain, config)) {
        return config
      }
    } catch (err) {
      // ignore transient read errors
    }
    await wait(500)
  }
  throw new Error('Timed out waiting for domain config change')
}

async function waitForDomainChange(field, expectedValue, description) {
  return waitForDomainUpdate(domain => {
    const current = JSON.stringify(domain[field] ?? null)
    return current === JSON.stringify(expectedValue)
  }, description)
}

async function waitForDomainPresence(name, shouldExist) {
  const start = Date.now()
  while (Date.now() - start < 30000) {
    const config = await readAgentConfig()
    const names = (config.domains || []).map(item => item.name)
    const exists = names.includes(name)
    if (exists === shouldExist) {
      return config
    }
    await wait(500)
  }
  throw new Error(`Timed out waiting for domain ${name} presence=${shouldExist}`)
}

async function waitForUpstreamTarget(upstreamKey, addr, shouldExist) {
  const start = Date.now()
  while (Date.now() - start < 30000) {
    const config = await readAgentConfig()
    const upstream = (config.upstreams || []).find(item => item.id === upstreamKey)
    const targets = upstream ? upstream.targets || [] : []
    const exists = targets.some(item => item.addr === addr)
    if (exists === shouldExist) {
      return config
    }
    await wait(500)
  }
  throw new Error(`Timed out waiting for upstream ${upstreamKey} target ${addr} presence=${shouldExist}`)
}

async function curlHost(extraArgs = []) {
  const args = [
    '-s',
    '-o', '/dev/null',
    '-w', '%{http_code}',
    '-H', `Host: ${domainHost}`,
    `http://${originHost}:${edgePort}/`,
    ...extraArgs
  ]
  return new Promise((resolve, reject) => {
    execFile('curl', args, { timeout: 30000 }, (err, stdout) => {
      if (err) return reject(err)
      resolve(stdout.trim())
    })
  })
}

async function getDomainConfig(config) {
  const entry = (config.domains || []).find(item => item.name === domainHost)
  if (!entry) {
    throw new Error(`Domain ${domainHost} not found in agent config`)
  }
  return entry
}

test.beforeEach(async ({ page }) => {
  page.setDefaultTimeout(30000)
  page.setDefaultNavigationTimeout(30000)
  await login(page)
})

test('manage basic + origin tabs sync', async ({ page }) => {
  test.setTimeout(300000)

  logStep('open manage page')
  await gotoManage(page)

  const baseConfig = await readAgentConfig()
  const baseVersion = baseConfig.version

  logStep('basic: toggle status')
  const statusSwitch = page.locator('.basic-config .el-form-item').filter({ hasText: '状态' }).locator('.el-switch').first()
  const statusChecked = await statusSwitch.evaluate(el => el.classList.contains('is-checked'))
  await waitForSave(page, () => statusSwitch.click())
  const statusConfig = await waitForDomainUpdate(domain => domain.status && domain.status !== 'active')
  const statusDomain = await getDomainConfig(statusConfig)
  expect(typeof statusDomain.status).toBe('string')
  await waitForSave(page, () => statusSwitch.click())
  await waitForDomainUpdate(domain => domain.status === 'active')

  logStep('basic: http enable toggle')
  const httpSection = page.locator('.basic-config .section-title', { hasText: 'HTTP设置' })
  await httpSection.scrollIntoViewIfNeeded()
  const httpSwitch = httpSection.locator('xpath=following-sibling::form[1]').locator('.el-switch').first()
  await waitForSave(page, () => httpSwitch.click())
  const httpConfig = await waitForDomainUpdate(domain => !Array.isArray(domain.http_listen) || domain.http_listen.length === 0)
  const httpDomain = await getDomainConfig(httpConfig)
  expect(httpDomain.http_listen === undefined || Array.isArray(httpDomain.http_listen)).toBeTruthy()
  await waitForSave(page, () => httpSwitch.click())
  await waitForDomainUpdate(domain => Array.isArray(domain.http_listen) && domain.http_listen.length > 0)

  logStep('basic: domain input')
  const domainLabel = page.locator('.basic-config .el-form-item__label', { hasText: '域名' }).first()
  const domainItem = domainLabel.locator('xpath=..')
  const domainInput = domainItem.locator('input.el-input__inner').first()
  const domainValue = await domainInput.inputValue()
  const domainTokens = domainValue.split(' ').filter(Boolean)
  const tempDomain = domainTokens.includes('testabc-tmp.665305.cc')
    ? domainTokens[0]
    : 'testabc-tmp.665305.cc'
  const nextDomainValue = domainTokens.includes('testabc-tmp.665305.cc')
    ? domainTokens.filter(d => d !== 'testabc-tmp.665305.cc').join(' ')
    : `${domainValue} ${tempDomain}`.trim()
  await waitForSave(page, async () => {
    await domainInput.fill(nextDomainValue)
    await domainInput.blur()
  })
  await waitForDomainPresence(tempDomain, !domainTokens.includes('testabc-tmp.665305.cc'))
  await waitForSave(page, async () => {
    await domainInput.fill(domainValue)
    await domainInput.blur()
  })
  await waitForDomainPresence(tempDomain, false)

  logStep('basic: http ports update')
  const httpPortsInput = httpSection.locator('xpath=following-sibling::form[1]').locator('input.el-input__inner').nth(0)
  const httpPortsValue = await httpPortsInput.inputValue()
  let basePorts = httpPortsValue.split(' ').filter(Boolean).filter(p => p !== '80')
  if (!basePorts.includes('81')) {
    basePorts.push('81')
  }
  if (basePorts.length === 0) {
    basePorts = ['81']
  }
  const baseHttpPorts = basePorts.join(' ')
  if (httpPortsValue.trim() !== baseHttpPorts) {
    await waitForSave(page, async () => {
      await httpPortsInput.fill(baseHttpPorts)
      await httpPortsInput.blur()
    })
    await waitForDomainChange('http_listen', basePorts, 'http_listen base')
  }
  const has82 = basePorts.includes('82')
  const nextPorts = has82 ? basePorts.filter(p => p !== '82') : [...basePorts, '82']
  const nextHttpPorts = nextPorts.join(' ')
  await waitForSave(page, async () => {
    await httpPortsInput.fill(nextHttpPorts)
    await httpPortsInput.blur()
  })
  const portsConfig = await waitForDomainChange('http_listen', nextPorts, 'http_listen update')
  const portsDomain = await getDomainConfig(portsConfig)
  if (!has82) {
    expect(portsDomain.http_listen.join(' ')).toContain('82')
  }
  await waitForSave(page, async () => {
    await httpPortsInput.fill(baseHttpPorts)
    await httpPortsInput.blur()
  })
  await waitForDomainChange('http_listen', basePorts, 'http_listen restore')

  logStep('basic: origin list add/remove')
  await page.getByRole('button', { name: '新增源站' }).scrollIntoViewIfNeeded()
  await waitForSave(page, () => page.getByRole('button', { name: '新增源站' }).click())
  const originTable = page.locator('.basic-config').locator('.el-table').first()
  const newOriginRow = originTable.locator('tbody tr').last()
  const tempOrigin = `${originHost}:8081`
  const originAddrInput = newOriginRow.locator('input').first()
  await waitForSave(page, async () => {
    await originAddrInput.fill(tempOrigin)
    await originAddrInput.blur()
  })
  const originConfig = await waitForUpstreamTarget(`upstream_${siteId}`, tempOrigin, true)
  const originDomain = await getDomainConfig(originConfig)
  expect(originDomain.upstream_key).toBeTruthy()
  await waitForSave(page, () => newOriginRow.getByRole('button', { name: '删除' }).click())
  await waitForUpstreamTarget(`upstream_${siteId}`, tempOrigin, false)

  logStep('basic: condition origin add/remove')
  await page.getByRole('button', { name: '新增条件源站' }).scrollIntoViewIfNeeded()
  await waitForSave(page, () => page.getByRole('button', { name: '新增条件源站' }).click())
  const conditionSection = page.locator('.basic-config .section-title', { hasText: '条件源站' })
  const conditionTable = conditionSection.locator('xpath=following-sibling::div[1]//table')
  const conditionRow = conditionTable.locator('tbody tr').last()
  await conditionRow.locator('.el-select').first().click()
  const condOpt = page.locator('.el-select-dropdown__item').filter({ hasText: '请求URI' }).first()
  if (await condOpt.count()) {
    await condOpt.click()
  } else {
    await page.keyboard.press('Escape')
  }
  const condOriginInput = conditionRow.locator('input[placeholder="源站地址，多个用 | 分隔"]')
  await waitForSave(page, async () => {
    await condOriginInput.fill(originHost)
    await condOriginInput.blur()
  })
  await waitForSave(page, () => conditionRow.getByRole('button', { name: '删除' }).click())

  logStep('origin tab: switch protocol + host + timeouts')
  await page.getByRole('tab', { name: '回源设置' }).click()
  const protocolGroup = page.locator('.origin-config .el-radio-group').first()
  const followChanged = await selectRadioIfNeeded(page, protocolGroup.getByText('跟随协议'))
  const protoConfig = followChanged ? await waitForAgentVersion(originConfig.version) : await readAgentConfig()
  const protoDomain = await getDomainConfig(protoConfig)
  expect(protoDomain.origin_protocol).toBeTruthy()
  const httpChanged = await selectRadioIfNeeded(page, protocolGroup.getByText('HTTP', { exact: true }))
  if (httpChanged) {
    await waitForDomainChange('origin_protocol', 'http', 'origin protocol http')
  }

  const httpPortLabel = page.locator('.origin-config .el-form-item__label', { hasText: 'HTTP回源端口' }).first()
  const httpPortInput = httpPortLabel.locator('xpath=..').locator('input.el-input__inner').first()
  const httpPortValue = await httpPortInput.inputValue()
  if (httpPortValue.trim() !== '80') {
    await waitForSave(page, async () => {
      await httpPortInput.fill('80')
      await httpPortInput.blur()
    })
    await waitForDomainChange('origin_http_port', '80', 'origin http port restore')
  }

  const hostGroup = page.locator('.origin-config .el-radio-group').nth(1)
  await selectRadioIfNeeded(page, hostGroup.getByText('自定义'))
  const hostInput = page.locator('.origin-config input[placeholder="请输入自定义回源HOST"]')
  await waitForSave(page, async () => {
    await hostInput.fill(domainHost)
    await hostInput.blur()
  })
  await selectRadioIfNeeded(page, hostGroup.getByText('自动跟随'))

  const timeoutLabel = page.locator('.origin-config .el-form-item__label', { hasText: '回源超时' }).first()
  const timeoutInput = timeoutLabel.locator('xpath=..').locator('input.el-input__inner').first()
  const timeoutVal = await timeoutInput.inputValue()
  await waitForSave(page, async () => {
    await timeoutInput.fill(String(Number(timeoutVal || '60') + 1))
    await timeoutInput.blur()
  })
  const connTimeoutLabel = page.locator('.origin-config .el-form-item__label', { hasText: '连接超时' }).first()
  const connTimeoutInput = connTimeoutLabel.locator('xpath=..').locator('input.el-input__inner').first()
  const connTimeoutVal = await connTimeoutInput.inputValue()
  await waitForSave(page, async () => {
    await connTimeoutInput.fill(String(Number(connTimeoutVal || '10') + 1))
    await connTimeoutInput.blur()
  })

  logStep('runtime: verify origin reachable with curl')
  const httpCode = await curlHost()
  expect(['200', '301', '302', '403']).toContain(httpCode)
})

test('cert upload + https tab sync', async ({ page }) => {
  test.setTimeout(300000)

  const certPem = fs.readFileSync(certPath, 'utf-8')
  const keyPem = fs.readFileSync(keyPath, 'utf-8')
  const certName = `auto-upload-${Date.now()}`

  logStep('certs: upload new cert')
  await page.goto('/website/certs')
  await page.waitForSelector('.app-container', { timeout: 30000 })
  await page.getByRole('button', { name: '添加证书' }).click()

  const dialog = page.locator('.el-dialog').filter({ hasText: '添加证书' })
  await dialog.waitFor({ state: 'visible', timeout: 30000 })

  const nameInput = dialog.locator('.el-form-item__label', { hasText: '名称' }).locator('xpath=..').locator('input')
  await nameInput.fill(certName)

  const certInput = dialog.locator('.el-form-item__label', { hasText: '证书' }).locator('xpath=..').locator('textarea')
  const keyInput = dialog.locator('.el-form-item__label', { hasText: '密钥' }).locator('xpath=..').locator('textarea')
  await certInput.fill(certPem)
  await keyInput.fill(keyPem)

  await Promise.all([
    page.waitForResponse(
      resp => resp.url().includes('/certs') && resp.request().method() === 'POST',
      { timeout: 30000 }
    ),
    dialog.getByRole('button', { name: '确定' }).click()
  ])
  await dialog.waitFor({ state: 'hidden', timeout: 30000 })

  const certRow = page.locator('tr').filter({ hasText: certName })
  await certRow.first().waitFor({ state: 'visible', timeout: 30000 })

  logStep('certs: verify edit shows cert/key')
  await certRow.first().locator('.link-type').click()
  const editDialog = page.locator('.el-dialog').filter({ hasText: '编辑证书' })
  await editDialog.waitFor({ state: 'visible', timeout: 30000 })
  const editCertInput = editDialog.locator('.el-form-item__label', { hasText: '证书' }).locator('xpath=..').locator('textarea')
  const editKeyInput = editDialog.locator('.el-form-item__label', { hasText: '密钥' }).locator('xpath=..').locator('textarea')
  expect(await editCertInput.inputValue()).toContain('BEGIN CERTIFICATE')
  expect(await editKeyInput.inputValue()).toContain('BEGIN PRIVATE KEY')
  await editDialog.getByRole('button', { name: '取消' }).click()
  await editDialog.waitFor({ state: 'hidden', timeout: 30000 })

  logStep('https: enable with uploaded cert')
  await gotoManage(page)
  await page.getByRole('tab', { name: 'HTTPS配置' }).click()

  const certSelect = page.locator('.https-config .el-select').first()
  await waitForSave(page, async () => {
    await certSelect.click()
    await page.locator('.el-select-dropdown__item').filter({ hasText: certName }).first().click()
  })

  const enableSwitch = page.locator('.https-config .el-form-item').filter({ hasText: '开关' }).first().locator('.el-switch')
  await ensureSwitch(page, enableSwitch, true)

  const listenLabel = page.locator('.https-config .el-form-item__label', { hasText: '监听端口' }).first()
  const listenInput = listenLabel.locator('xpath=..').locator('input.el-input__inner').first()
  const listenValue = await listenInput.inputValue()
  if (listenValue.trim() !== '4443') {
    await waitForSave(page, async () => {
      await listenInput.fill('4443')
      await listenInput.blur()
    })
  }
  await waitForDomainChange('https_listen', ['4443'], 'https_listen update')

  const forceSection = page.locator('.https-config .section-title', { hasText: '强制HTTPS' })
  const forceSwitch = forceSection.locator('xpath=following-sibling::*[1]').locator('.el-switch')
  await ensureSwitch(page, forceSwitch, true)
  await waitForDomainChange('https_force', true, 'https force enable')

  const forcePortSelect = page.locator('.https-config .el-form-item__label', { hasText: '跳转端口' }).locator('xpath=..').locator('.el-select')
  if ((await forcePortSelect.count()) > 0) {
    await forcePortSelect.click()
    await page.locator('.el-select-dropdown__item').filter({ hasText: '4443' }).first().click()
  }
  await waitForDomainChange('https_redirect_port', '4443', 'https redirect port')

  const hstsSection = page.locator('.https-config .section-title', { hasText: 'HSTS' })
  await ensureSwitch(page, hstsSection.locator('xpath=following-sibling::*[1]').locator('.el-switch'), true)
  await waitForDomainChange('https_hsts', true, 'https hsts')

  const http2Section = page.locator('.https-config .section-title', { hasText: 'HTTP2设置' })
  await ensureSwitch(page, http2Section.locator('xpath=following-sibling::*[1]').locator('.el-switch'), true)
  await waitForDomainChange('https_http2', true, 'https http2')

  const ocspSection = page.locator('.https-config .section-title', { hasText: 'OCSP Stapling' })
  await ensureSwitch(page, ocspSection.locator('xpath=following-sibling::*[1]').locator('.el-switch'), true)
  await waitForDomainChange('https_ocsp', true, 'https ocsp')

  const http3Section = page.locator('.https-config .section-title', { hasText: 'HTTP3设置' })
  await ensureSwitch(page, http3Section.locator('xpath=following-sibling::*[1]').locator('.el-switch'), true)
  await waitForDomainChange('https_http3', true, 'https http3')

  const sslPolicyGroup = page.locator('.https-config .el-radio-group').last()
  await selectRadioIfNeeded(page, sslPolicyGroup.getByText('自定义'))

  const ciphersLabel = page.locator('.https-config .el-form-item__label', { hasText: '加密算法' }).first()
  const ciphersInput = ciphersLabel.locator('xpath=..').locator('textarea')
  if ((await ciphersInput.count()) && (await ciphersInput.isVisible())) {
    const ciphersDisabled = await ciphersInput.evaluate(el => el.hasAttribute('disabled'))
    if (!ciphersDisabled) {
      const targetCiphers = 'ECDHE-RSA-AES128-GCM-SHA256'
      const currentCiphers = await ciphersInput.evaluate(el => el.value || '')
      if (currentCiphers.trim() !== targetCiphers) {
        await waitForSave(page, async () => {
          await ciphersInput.fill(targetCiphers)
          await ciphersInput.blur()
        })
        await waitForDomainChange('https_ssl_ciphers', targetCiphers, 'https ssl ciphers')
      }
    }
  }

  const protocolsLabel = page.locator('.https-config .el-form-item__label', { hasText: 'SSL协议' }).first()
  const protocolsInput = protocolsLabel.locator('xpath=..').locator('textarea')
  if ((await protocolsInput.count()) && (await protocolsInput.isVisible())) {
    const protocolsDisabled = await protocolsInput.evaluate(el => el.hasAttribute('disabled'))
    if (!protocolsDisabled) {
      const targetProtocols = 'TLSv1.2 TLSv1.3'
      const currentProtocols = await protocolsInput.evaluate(el => el.value || '')
      if (currentProtocols.trim() !== targetProtocols) {
        await waitForSave(page, async () => {
          await protocolsInput.fill(targetProtocols)
          await protocolsInput.blur()
        })
        await waitForDomainChange('https_ssl_protocols', targetProtocols, 'https ssl protocols')
      }
    }
  }
})
