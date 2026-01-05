<template>
  <el-tabs v-model="activeTab" v-loading="loading" class="sub-tabs" @tab-change="handleTabChange">
    <el-tab-pane label="网站默认配置" name="website">
      <config-form v-if="activeTab === 'website'" type="website" :data="defaults.website" @change="save('website')" />
    </el-tab-pane>
    <el-tab-pane label="API 默认配置" name="api">
      <config-form v-if="activeTab === 'api'" type="api" :data="defaults.api" @change="save('api')" />
    </el-tab-pane>
    <el-tab-pane label="下载默认配置" name="download">
      <config-form v-if="activeTab === 'download'" type="download" :data="defaults.download" @change="save('download')" />
    </el-tab-pane>
  </el-tabs>
</template>

<script setup>
import { reactive, ref, onMounted, defineComponent, h } from 'vue'
import { ElMessage, ElForm, ElFormItem, ElSwitch, ElInput } from 'element-plus'
import request from '@/utils/request'

const activeTab = ref('website')
const loading = ref(false)
const globalConfig = ref(null)

const defaults = reactive({
  website: { cache_enable: true, cache_ttl: 86400, gzip: true, waf_enable: true },
  api: { cache_enable: false, cache_ttl: 0, gzip: true, waf_enable: false },
  download: { cache_enable: true, cache_ttl: 86400, gzip: true, waf_enable: true }
})

const ConfigForm = defineComponent({
  props: ['type', 'data'],
  emits: ['change'],
  setup(props, { emit }) {
    return () => h(ElForm, { labelWidth: '150px', class: 'config-form' }, {
      default: () => [
        h(ElFormItem, { label: '缓存开关', style: { maxWidth: '500px' } }, {
          default: () => h(ElSwitch, { 
            modelValue: props.data.cache_enable,
            'onUpdate:modelValue': (val) => { props.data.cache_enable = val; emit('change') }
          })
        }),
        h(ElFormItem, { label: '缓存 TTL (秒)', style: { maxWidth: '500px' } }, {
          default: () => h(ElInput, { 
            modelValue: String(props.data.cache_ttl ?? ''),
            'onUpdate:modelValue': (val) => { props.data.cache_ttl = Number(val) || 0; emit('change') }
          })
        }),
        h(ElFormItem, { label: '开启 Gzip', style: { maxWidth: '500px' } }, {
          default: () => h(ElSwitch, { 
            modelValue: props.data.gzip,
            'onUpdate:modelValue': (val) => { props.data.gzip = val; emit('change') }
          })
        }),
        h(ElFormItem, { label: '开启 WAF', style: { maxWidth: '500px' } }, {
          default: () => h(ElSwitch, { 
            modelValue: props.data.waf_enable,
            'onUpdate:modelValue': (val) => { props.data.waf_enable = val; emit('change') }
          })
        })
      ]
    })
  }
})

const save = async (mode) => {
  try {
    if (!globalConfig.value) {
      await load()
    }
    const cfg = globalConfig.value || {}
    if (!cfg.default_config) {
      cfg.default_config = { website: {}, api: {}, download: {} }
    }
    cfg.default_config[mode] = { ...defaults[mode] }
    await request.post('/global_config', cfg)
    ElMessage.success('保存成功')
  } catch (err) {
    ElMessage.error('保存失败')
  }
}

const applyDefaultConfig = (cfg) => {
  const def = cfg?.default_config || {}
  if (def.website) Object.assign(defaults.website, def.website)
  if (def.api) Object.assign(defaults.api, def.api)
  if (def.download) Object.assign(defaults.download, def.download)
}

const load = async () => {
  loading.value = true
  try {
    const res = await request.get('/global_config')
    globalConfig.value = res.data || {}
    applyDefaultConfig(globalConfig.value)
  } finally {
    loading.value = false
  }
}

const handleTabChange = () => {
  load()
}

onMounted(() => {
  load()
})
</script>
