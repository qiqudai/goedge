<template>
  <div class="app-container" @focusin="cacheInputValue">
    <ErrorPageLangSettings
      :model-value="errorPageI18n"
      @update:model-value="updateErrorPageI18n"
      @save="saveConfig"
    />

    <el-card>
      <template #header>
        <div class="card-header">
          <span>自定义错误页面</span>
          <el-button type="primary" :loading="saving" @click="saveConfig">保存配置</el-button>
        </div>
      </template>

      <div class="error-page-container">
        <el-tabs v-model="activeCode" tab-position="left" class="error-tabs" style="height: calc(100vh - 360px);">
          <el-tab-pane v-for="code in errorCodes" :key="code.key" :label="code.label" :name="code.key" lazy>
            <div class="tab-content-scroll">
              <div class="editor-header">
                <h3>{{ code.label }} ({{ code.key }})</h3>
              </div>

              <el-tabs v-model="innerTabs[code.key]" class="inner-tabs">
                <el-tab-pane label="HTML 模板" name="template" lazy>
                  <ErrorPageTemplateEditor v-model="pageDef(code.key).template" />
                </el-tab-pane>
                <el-tab-pane label="多语言文案" name="strings" lazy>
                  <ErrorPageTranslationTable
                    :template="pageDef(code.key).template"
                    :strings="pageDef(code.key).strings"
                    :enabled-langs="errorPageI18n.enabled_langs"
                    @update:strings="value => updateStrings(code.key, value)"
                  />
                </el-tab-pane>
                <el-tab-pane label="预览" name="preview" lazy>
                  <div class="preview-toolbar">
                    <span>预览语言</span>
                    <el-select v-model="previewLang[code.key]" style="width: 200px">
                      <el-option
                        v-for="lang in errorPageI18n.enabled_langs"
                        :key="lang"
                        :label="lang"
                        :value="lang"
                      />
                    </el-select>
                  </div>
                  <div class="preview" v-html="previewHtml(code.key)"></div>
                </el-tab-pane>
              </el-tabs>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { defineAsyncComponent, reactive, ref, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { cacheInputValue } from '@/utils/saveGuard'
import { fetchGlobalConfig, saveGlobalConfig } from '@/api/globalConfig'
import { ERROR_PAGE_CODES, DEFAULT_ERROR_PAGE_I18N } from '@/constants/errorPageLocales'
import {
  buildGlobalConfigPayload,
  ensureErrorPageStructure,
  fillMissingErrorPageStrings,
  renderErrorPagePreview,
  resolvePreviewStrings
} from '@/services/errorPageService'
import ErrorPageLangSettings from '@/components/global/ErrorPageLangSettings.vue'
import ErrorPageTemplateEditor from '@/components/global/ErrorPageTemplateEditor.vue'

const ErrorPageTranslationTable = defineAsyncComponent(() => import('@/components/global/ErrorPageTranslationTable.vue'))

const loading = ref(false)
const saving = ref(false)
const activeCode = ref('403')
const fullConfig = ref({})
const errorPages = reactive({})
const errorPageI18n = reactive({
  default_lang: 'zh-CN',
  lang_mode: 'browser',
  enabled_langs: [...DEFAULT_ERROR_PAGE_I18N.enabled_langs]
})
const innerTabs = reactive({})
const previewLang = reactive({})
const errorCodes = ERROR_PAGE_CODES

const pageDef = (code) => errorPages[code] || { template: '', strings: {} }

const initTabState = () => {
  errorCodes.forEach(code => {
    if (!innerTabs[code.key]) innerTabs[code.key] = 'template'
    if (!previewLang[code.key]) previewLang[code.key] = errorPageI18n.default_lang
  })
}

const applyStructure = (pages, i18n) => {
  const normalized = ensureErrorPageStructure(pages, i18n)
  Object.keys(errorPages).forEach(key => delete errorPages[key])
  Object.assign(errorPages, normalized.pages)
  Object.assign(errorPageI18n, normalized.i18n)
  initTabState()
}

const updateErrorPageI18n = (value) => {
  Object.assign(errorPageI18n, value)
}

const loadConfig = async () => {
  loading.value = true
  try {
    const res = await fetchGlobalConfig()
    if (res.code === 0 || res.code === 200) {
      fullConfig.value = res.data || {}
      applyStructure(res.data?.error_pages, res.data?.error_page_i18n)
    }
  } finally {
    loading.value = false
  }
}

const updateStrings = (code, value) => {
  if (!errorPages[code]) return
  errorPages[code].strings = value
}

const previewHtml = (code) => {
  const def = errorPages[code]
  if (!def) return ''
  const lang = previewLang[code] || errorPageI18n.default_lang
  const strings = resolvePreviewStrings(def, lang, errorPageI18n.default_lang)
  return renderErrorPagePreview(def.template, strings)
    .replaceAll('{client_ip}', '203.0.113.1')
    .replaceAll('{node_ip}', '198.51.100.1')
}

const saveConfig = async () => {
  saving.value = true
  try {
    const payload = buildGlobalConfigPayload(fullConfig.value, errorPages, errorPageI18n)
    const res = await saveGlobalConfig(payload)
    if (res.code === 0 || res.code === 200) {
      ElMessage.success('保存成功')
      fullConfig.value = payload
    }
  } finally {
    saving.value = false
  }
}

watch(
  () => [...errorPageI18n.enabled_langs],
  langs => {
    errorCodes.forEach(({ key }) => {
      if (!langs.includes(previewLang[key])) {
        previewLang[key] = errorPageI18n.default_lang
      }
      if (!errorPages[key]) return
      errorPages[key].strings = fillMissingErrorPageStrings(key, errorPages[key].strings, langs)
    })
  }
)

onMounted(() => {
  applyStructure({}, DEFAULT_ERROR_PAGE_I18N)
  loadConfig()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.error-tabs {
  border: 1px solid var(--el-border-color-light);
}
:deep(.el-tabs__content) {
  height: 100%;
  overflow-y: auto;
  padding-right: 10px;
}
.tab-content-scroll {
  padding-bottom: 20px;
}
.editor-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.preview-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.preview {
  margin-top: 12px;
  border: 1px dashed var(--el-border-color);
  padding: 10px;
  background: var(--el-fill-color-light);
  min-height: 100px;
}
</style>
