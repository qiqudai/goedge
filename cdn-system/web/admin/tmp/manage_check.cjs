const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const base = 'https://cccadmin.665305.cc';
  const siteId = 37;

  const waitForSave = () => page.waitForResponse((resp) => {
    return resp.url().includes(`/api/v1/admin/sites/${siteId}`) && resp.request().method() === 'PUT' && resp.status() === 200;
  }, { timeout: 5000 });

  const trySave = async (action, label) => {
    const waiter = waitForSave().catch(() => null);
    await action();
    const resp = await waiter;
    if (!resp) {
      console.log(`[Warn] save response not observed: ${label}`);
    }
  };

  const clickTab = async (name) => {
    console.log(`[Tab] ${name}`);
    await page.getByRole('tab', { name }).click();
  };

  const formItemByLabel = (scope, label) => {
    return scope.locator('.el-form-item', { has: scope.locator('.el-form-item__label', { hasText: label }) }).first();
  };

  const escapeRegex = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const clickRadio = (scope, label) => {
    const re = new RegExp(`^${escapeRegex(label)}$`);
    return scope.locator('.el-radio', { hasText: re }).click();
  };

  try {
    console.log('[Login]');
    await page.goto(`${base}/login`, { waitUntil: 'domcontentloaded' });
    await page.getByPlaceholder('Username').fill('admin');
    await page.getByPlaceholder('Password').fill('123456');
    await page.getByRole('button', { name: 'Login' }).click();
    await page.waitForURL('**/dashboard', { timeout: 30000 });

    console.log('[Navigate] manage');
    await page.goto(`${base}/website/manage?site_id=${siteId}`, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('.manage-tabs', { timeout: 30000 });

    const basicScope = page.locator('.basic-config');
    const originScope = page.locator('.origin-config');
    const httpsScope = page.locator('.https-config');
    const securityScope = page.locator('.security-config');
    const cacheScope = page.locator('.cache-config');
    const accessScope = page.locator('.access-config');
    const advancedScope = page.locator('.advanced-config');

    // Basic tab
    await clickTab('基本配置');
    console.log('[Basic] toggle status');
    const statusSwitch = basicScope.locator('.el-switch').first();
    await trySave(() => statusSwitch.click(), 'basic.status.on');
    await trySave(() => statusSwitch.click(), 'basic.status.off');

    console.log('[Basic] add/remove origin');
    const originTable = basicScope.locator('.el-table').first();
    const originRows = originTable.locator('.el-table__body tr');
    const originCount = await originRows.count();
    await page.getByRole('button', { name: '新增源站' }).click();
    await page.waitForFunction((count) => {
      return document.querySelectorAll('.basic-config .el-table').length > 0 &&
        document.querySelectorAll('.basic-config .el-table')[0].querySelectorAll('tbody tr').length > count;
    }, originCount);
    const lastOrigin = originRows.last();
    await trySave(() => lastOrigin.locator('.el-switch').click(), 'basic.origin.toggle');
    await trySave(() => lastOrigin.getByRole('button', { name: '删除' }).click(), 'basic.origin.remove');

    console.log('[Basic] add/remove condition origin');
    const conditionTable = basicScope.locator('.el-table').nth(1);
    const condRows = conditionTable.locator('.el-table__body tr');
    const condCount = await condRows.count();
    await page.getByRole('button', { name: '新增条件源站' }).click();
    await page.waitForFunction((count) => {
      return document.querySelectorAll('.basic-config .el-table').length > 1 &&
        document.querySelectorAll('.basic-config .el-table')[1].querySelectorAll('tbody tr').length > count;
    }, condCount);
    const lastCond = condRows.last();
    await trySave(() => lastCond.getByRole('button', { name: '删除' }).click(), 'basic.originCondition.remove');

    // Origin tab
    await clickTab('回源设置');
    console.log('[Origin] protocol + host');
    await trySave(() => clickRadio(originScope, 'HTTP'), 'origin.protocol.http');
    await trySave(() => clickRadio(originScope, '跟随协议'), 'origin.protocol.follow');
    await trySave(() => clickRadio(originScope, '自定义'), 'origin.host.custom');
    const hostInput = originScope.getByPlaceholder('请输入自定义回源HOST');
    await hostInput.fill('202.155.141.95');
    await trySave(() => hostInput.press('Tab'), 'origin.host.value');

    // HTTPS tab
    await clickTab('HTTPS配置');
    console.log('[HTTPS] toggle http2');
    const http2Switch = httpsScope.locator('.section-title', { hasText: 'HTTP2设置' })
      .locator('xpath=following-sibling::div[contains(@class,"el-form-item")][1]')
      .locator('.el-switch');
    if (await http2Switch.count()) {
      await trySave(() => http2Switch.click(), 'https.http2.on');
      await trySave(() => http2Switch.click(), 'https.http2.off');
    } else {
      console.log('[Warn] HTTP2 switch not found');
    }

    // Security tab
    await clickTab('安全设置');
    console.log('[Security] crawler');
    const crawlerForm = securityScope.locator('.section-title', { hasText: '搜索引擎爬虫' })
      .locator('xpath=following-sibling::form[1]');
    await trySave(() => clickRadio(crawlerForm, '放行'), 'security.crawler.allow');
    await trySave(() => clickRadio(crawlerForm, '不设置'), 'security.crawler.none');

    console.log('[Security] auto switch');
    const autoSwitchItem = formItemByLabel(securityScope, '自动切换');
    const autoSwitchToggle = autoSwitchItem.locator('.el-switch');
    if (await autoSwitchToggle.count()) {
      await trySave(() => autoSwitchToggle.click(), 'security.autoSwitch.on');
      await trySave(() => autoSwitchToggle.click(), 'security.autoSwitch.off');
    } else {
      console.log('[Warn] autoSwitch toggle not found');
    }

    console.log('[Security] custom rule');
    await securityScope.getByRole('button', { name: '新增规则' }).click();
    const ruleDialog = page.locator('.el-dialog').filter({ hasText: '新增规则' }).first();
    await ruleDialog.waitFor({ state: 'visible', timeout: 30000 });
    await ruleDialog.getByRole('button', { name: '确定' }).click();
    await waitForSave().catch(() => console.log('[Warn] save response not observed: security.rule.add'));
    const ruleTable = securityScope.locator('.el-table__body');
    if (await ruleTable.getByRole('button', { name: '删除' }).count()) {
      await trySave(() => ruleTable.getByRole('button', { name: '删除' }).first().click(), 'security.rule.remove');
    }

    console.log('[Security] cookie + proxy + region');
    const cookieForm = securityScope.locator('.section-title', { hasText: '设置Cookie域名' })
      .locator('xpath=following-sibling::form[1]');
    const cookieSwitch = cookieForm.locator('.el-switch').first();
    if (await cookieSwitch.count()) {
      await trySave(() => cookieSwitch.click(), 'security.cookie.on');
      const cookieInput = securityScope.getByPlaceholder('例如: abc.com');
      if (await cookieInput.count()) {
        await cookieInput.fill('example.com');
        await trySave(() => cookieInput.press('Tab'), 'security.cookie.value');
      } else {
        console.log('[Warn] cookie input not found');
      }
      await trySave(() => cookieSwitch.click(), 'security.cookie.off');
    } else {
      console.log('[Warn] cookie switch not found');
    }

    const proxyForm = securityScope.locator('.section-title', { hasText: '屏蔽设置' })
      .locator('xpath=following-sibling::form[1]');
    const proxySwitch = proxyForm.locator('.el-switch').first();
    if (await proxySwitch.count()) {
      await trySave(() => proxySwitch.click(), 'security.proxy.on');
      await trySave(() => proxySwitch.click(), 'security.proxy.off');
    } else {
      console.log('[Warn] proxy switch not found');
    }

    const regionForm = securityScope.locator('.section-title', { hasText: '区域屏蔽' })
      .locator('xpath=following-sibling::form[1]');
    await trySave(() => clickRadio(regionForm, '中国（包括港澳台）'), 'security.region.china');
    await trySave(() => clickRadio(regionForm, '不设置'), 'security.region.none');

    // Cache tab
    await clickTab('缓存设置');
    console.log('[Cache] preset');
    await cacheScope.locator('.el-select').click();
    await page.getByRole('option', { name: '静态资源缓存' }).click();
    await waitForSave().catch(() => console.log('[Warn] save response not observed: cache.preset'));

    // Access tab
    await clickTab('访问控制');
    console.log('[Access] hotlink + cors');
    const hotlinkSwitch = formItemByLabel(accessScope, '开关').locator('.el-switch').first();
    if (await hotlinkSwitch.count()) {
      await trySave(() => hotlinkSwitch.click(), 'access.hotlink.on');
      await trySave(() => clickRadio(accessScope, '后缀'), 'access.hotlink.scope');
      const hotlinkValue = accessScope.getByPlaceholder('请输入后缀，如 png|jpg|gif');
      if (await hotlinkValue.count()) {
        await hotlinkValue.fill('jpg');
        await trySave(() => hotlinkValue.press('Tab'), 'access.hotlink.value');
      } else {
        console.log('[Warn] hotlink value input not found');
      }
      await trySave(() => hotlinkSwitch.click(), 'access.hotlink.off');
    } else {
      console.log('[Warn] hotlink switch not found');
    }

    const corsSwitch = accessScope.locator('.section-title', { hasText: '跨域访问设置' })
      .locator('xpath=following-sibling::div[contains(@class,"el-form-item")][1]')
      .locator('.el-switch');
    if (await corsSwitch.count()) {
      await trySave(() => corsSwitch.click(), 'access.cors.on');
      const moreToggle = accessScope.getByText('查看更多设置');
      if (await moreToggle.count()) {
        await moreToggle.click();
        const allowOrigin = formItemByLabel(accessScope, 'allow_origin').locator('input');
        if (await allowOrigin.count()) {
          await allowOrigin.fill('*');
          await trySave(() => allowOrigin.press('Tab'), 'access.cors.allowOrigin');
        } else {
          console.log('[Warn] allow_origin input not found');
        }
      } else {
        console.log('[Warn] CORS more toggle not found');
      }
      await trySave(() => corsSwitch.click(), 'access.cors.off');
    } else {
      console.log('[Warn] cors switch not found');
    }

    // Advanced tab
    await clickTab('高级设置');
    console.log('[Advanced] search engine origin + redirect');
    const searchSection = advancedScope.locator('.section-title', { hasText: '搜索引擎回源配置' })
      .locator('xpath=following-sibling::div[contains(@class,"el-form-item")][1]');
    const searchSwitch = searchSection.locator('.el-switch').first();
    if (await searchSwitch.count()) {
      await trySave(() => searchSwitch.click(), 'advanced.search.on');
      const originIpInput = advancedScope.getByPlaceholder('请输入源IP');
      await originIpInput.fill('202.155.141.95');
      await trySave(() => originIpInput.press('Tab'), 'advanced.search.value');
      await trySave(() => searchSwitch.click(), 'advanced.search.off');
    } else {
      console.log('[Warn] search engine switch not found');
    }

    await advancedScope.getByRole('button', { name: '新增转向' }).click();
    const redirectDialog = page.locator('.el-dialog').filter({ hasText: '新增转向' }).first();
    await redirectDialog.waitFor({ state: 'visible', timeout: 30000 });
    await redirectDialog.getByRole('button', { name: '确定' }).click();
    await waitForSave().catch(() => console.log('[Warn] save response not observed: advanced.redirect.add'));
    const redirectTable = advancedScope.locator('.el-table__body');
    if (await redirectTable.getByRole('button', { name: '删除' }).count()) {
      await trySave(() => redirectTable.getByRole('button', { name: '删除' }).first().click(), 'advanced.redirect.remove');
    }
  } catch (err) {
    console.error(err);
  } finally {
    await browser.close();
  }
})();

