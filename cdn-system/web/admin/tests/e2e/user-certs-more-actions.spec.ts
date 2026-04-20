import fs from 'node:fs'
import path from 'node:path'
import { expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

test('user: cert download and auto renew actions', async () => {
  test.setTimeout(120_000)

  const { token } = await loginUser('ceshi', '123456')
  const api = await createUserApiContext(token)

  const certPath = process.env.E2E_CERT_PEM || '/www/server/go_project/openresty/cdn-system/agent/edge-node/cert/fallback.pem'
  const keyPath = process.env.E2E_CERT_KEY || '/www/server/go_project/openresty/cdn-system/agent/edge-node/cert/fallback.key'
  const certFullPath = path.resolve(certPath)
  const keyFullPath = path.resolve(keyPath)
  if (!fs.existsSync(certFullPath) || !fs.existsSync(keyFullPath)) {
    test.skip(true, `cert fixture missing: ${certFullPath} ${keyFullPath}`)
  }
  const certPem = fs.readFileSync(certFullPath, 'utf-8')
  const keyPem = fs.readFileSync(keyFullPath, 'utf-8')

  const domain = `autotest-cert-actions-${Date.now()}.example.com`
  const uploadBody = await expectApiSuccess(
    await api.post('/api/v1/user/certs', {
      data: { name: `upload-${Date.now()}`, type: 'upload', domain, cert: certPem, key: keyPem }
    })
  )
  const uploadId = uploadBody.data?.id || uploadBody.id
  expect(uploadId).toBeTruthy()

  const acmeBody = await expectApiSuccess(
    await api.post('/api/v1/user/certs', {
      data: { name: `acme-${Date.now()}`, type: 'letsencrypt', domain, cert: certPem, key: keyPem }
    })
  )
  const acmeId = acmeBody.data?.id || acmeBody.id
  expect(acmeId).toBeTruthy()

  await expectApiSuccess(await api.post('/api/v1/user/certs/batch_action', { data: { action: 'auto_renew_disable', ids: [uploadId, acmeId] } }))

  let listBody = await expectApiSuccess(
    await api.get('/api/v1/user/certs', { params: { search_field: 'domain', keyword: domain, pageSize: 10 } })
  )
  const list = listBody.data?.list || listBody.list || listBody.data || []
  const uploadRow = list.find((item: any) => Number(item.id) === Number(uploadId))
  const acmeRow = list.find((item: any) => Number(item.id) === Number(acmeId))
  expect(uploadRow?.auto_renew).toBeFalsy()
  expect(acmeRow?.auto_renew).toBeFalsy()

  await expectApiSuccess(await api.post('/api/v1/user/certs/batch_action', { data: { action: 'auto_renew_enable', ids: [acmeId] } }))

  listBody = await expectApiSuccess(
    await api.get('/api/v1/user/certs', { params: { search_field: 'domain', keyword: domain, pageSize: 10 } })
  )
  const listAfter = listBody.data?.list || listBody.list || listBody.data || []
  const acmeAfter = listAfter.find((item: any) => Number(item.id) === Number(acmeId))
  expect(acmeAfter?.auto_renew).toBeTruthy()

  const downloadRes = await api.get(`/api/v1/user/certs/${uploadId}/download`, { params: { domain } })
  expect(downloadRes.ok()).toBeTruthy()
  const zipBuf = await downloadRes.body()
  expect(zipBuf.length).toBeGreaterThan(100)
  expect(zipBuf.slice(0, 2).toString()).toBe('PK')

  await expectApiSuccess(await api.post('/api/v1/user/certs/batch_action', { data: { action: 'disable', ids: [uploadId, acmeId] } }))
  await api.delete(`/api/v1/user/certs/${uploadId}`).catch(() => null)
  await api.delete(`/api/v1/user/certs/${acmeId}`).catch(() => null)

  await api.dispose()
})
