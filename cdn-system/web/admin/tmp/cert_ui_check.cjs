const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  const base = 'https://cccadmin.665305.cc';
  await page.goto(`${base}/login`, { waitUntil: 'domcontentloaded' });
  await page.getByPlaceholder('Username').fill('admin');
  await page.getByPlaceholder('Password').fill('123456');
  await page.getByRole('button', { name: 'Login' }).click();
  await page.waitForURL('**/dashboard', { timeout: 30000 });

  await page.goto(`${base}/website/certs`, { waitUntil: 'domcontentloaded' });
  await page.getByRole('button', { name: '添加证书' }).click();

  const dialog = page.locator('.el-dialog').first();
  await dialog.waitFor({ state: 'visible', timeout: 30000 });

  const uploadLabel = dialog.locator('.el-radio__label', { hasText: '自己上传' }).first();
  await uploadLabel.waitFor({ state: 'visible', timeout: 30000 });
  await uploadLabel.click();

  const certField = page.getByPlaceholder('-----BEGIN CERTIFICATE-----');
  const uploadVisible = await certField.isVisible();

  const letsLabel = dialog.locator('.el-radio__label', { hasText: "Let's Encrypt" }).first();
  await letsLabel.click();
  const letsVisible = await certField.isVisible();

  console.log(`[Add Dialog] upload cert visible: ${uploadVisible}, letsencrypt cert visible: ${letsVisible}`);

  const wildcardTab = dialog.locator('.el-tabs__item', { hasText: '泛证书申请' });
  await wildcardTab.click();
  const wildcardPane = dialog.locator('.el-tab-pane:visible');
  const manualVisible = await wildcardPane.locator('.dns-manual').isVisible();
  console.log(`[Wildcard Tab] manual block visible: ${manualVisible}`);

  await dialog.getByRole('button', { name: '取消' }).click();

  const firstRowLink = page.locator('.link-type').first();
  await firstRowLink.click();
  const editDialog = page.locator('.el-dialog').first();
  await editDialog.waitFor({ state: 'visible', timeout: 30000 });

  const editCertVisible = await certField.isVisible();
  const editUploadLabel = editDialog.locator('.el-radio__label', { hasText: '自己上传' }).first();
  await editUploadLabel.click();
  const editUploadVisible = await certField.isVisible();

  console.log(`[Edit Dialog] default cert visible: ${editCertVisible}, upload cert visible: ${editUploadVisible}`);

  await browser.close();
})();
