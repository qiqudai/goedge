<template>
  <el-card class="lang-settings-card" shadow="never">
    <template #header>
      <span>全局错误页语言</span>
    </template>
    <el-form label-width="140px">
      <el-form-item label="语言策略">
        <el-radio-group :model-value="modelValue.lang_mode" @update:model-value="updateField('lang_mode', $event)">
          <el-radio value="browser">跟随浏览器语言</el-radio>
          <el-radio value="fixed">固定默认语言</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="默认语言">
        <el-select
          :model-value="modelValue.default_lang"
          filterable
          allow-create
          default-first-option
          style="width: 280px"
          @update:model-value="updateDefaultLang"
        >
          <el-option
            v-for="item in localeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="启用语言">
        <el-select
          :model-value="modelValue.enabled_langs"
          multiple
          filterable
          allow-create
          default-first-option
          style="width: 100%; max-width: 720px"
          @update:model-value="updateEnabledLangs"
        >
          <el-option
            v-for="item in localeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
        <div class="form-tip">可搜索 BCP47 语言代码并添加，例如 zh-CN、en、ja</div>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup>
import { COMMON_ERROR_PAGE_LOCALES } from '@/constants/errorPageLocales'
import { normalizeLocaleTag } from '@/services/errorPageService'

const props = defineProps({
  modelValue: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['update:modelValue'])

const localeOptions = COMMON_ERROR_PAGE_LOCALES

const updateField = (key, value) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

const updateDefaultLang = (value) => {
  const lang = normalizeLocaleTag(value)
  const enabled = new Set(props.modelValue.enabled_langs || [])
  enabled.add(lang)
  emit('update:modelValue', {
    ...props.modelValue,
    default_lang: lang,
    enabled_langs: Array.from(enabled)
  })
}

const updateEnabledLangs = (values) => {
  const langs = (values || []).map(normalizeLocaleTag).filter(Boolean)
  const unique = Array.from(new Set(langs))
  if (!unique.includes(props.modelValue.default_lang)) {
    unique.unshift(props.modelValue.default_lang)
  }
  emit('update:modelValue', { ...props.modelValue, enabled_langs: unique })
}
</script>

<style scoped>
.lang-settings-card {
  margin-bottom: 16px;
}
.form-tip {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
