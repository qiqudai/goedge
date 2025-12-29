<template>
  <el-tabs v-model="activeTab" class="sub-tabs">
    <el-tab-pane label="网站默认配置" name="site">
      <config-form type="site" :data="defaults.site" @change="save('site')" />
    </el-tab-pane>
    <el-tab-pane label="API 默认配置" name="api">
      <config-form type="api" :data="defaults.api" @change="save('api')" />
    </el-tab-pane>
    <el-tab-pane label="下载默认配置" name="download">
      <config-form type="download" :data="defaults.download" @change="save('download')" />
    </el-tab-pane>
  </el-tabs>
</template>

<script setup>
import { reactive, ref, onMounted, defineComponent, h } from 'vue'
import { ElMessage, ElForm, ElFormItem, ElSwitch, ElInput } from 'element-plus'
import request from '@/utils/request'

const activeTab = ref('site')

const defaults = reactive({
  site: { enable: true, ttl: '86400', gzip: true, waf: true },
  api: { enable: false, ttl: '0', gzip: true, waf: false },
  download: { enable: true, ttl: '86400', gzip: true, waf: true }
})

const ConfigForm = defineComponent({
  props: ['type', 'data'],
  emits: ['change'],
  setup(props, { emit }) {
    return () => h(ElForm, { labelWidth: '150px', class: 'config-form' }, {
      default: () => [
        h(ElFormItem, { label: '缓存开关', style: { maxWidth: '500px' } }, {
          default: () => h(ElSwitch, { 
            modelValue: props.data.enable, 
            'onUpdate:modelValue': (val) => { props.data.enable = val; emit('change') }
          })
        }),
        h(ElFormItem, { label: '缓存 TTL (秒)', style: { maxWidth: '500px' } }, {
          default: () => h(ElInput, { 
            modelValue: props.data.ttl, 
            'onUpdate:modelValue': (val) => { props.data.ttl = val; emit('change') }
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
            modelValue: props.data.waf, 
            'onUpdate:modelValue': (val) => { props.data.waf = val; emit('change') }
          })
        })
      ]
    })
  }
})

const save = async (mode) => {
  try {
    await request.post('/global_config', {
      name: `cache_default_${mode}`,
      value: JSON.stringify(defaults[mode])
    })
    ElMessage.success('保存成功')
  } catch (err) {
    ElMessage.error('保存失败')
  }
}

const load = async () => {
  const res = await request.get('/global_config')
  const items = res?.data?.data || []
  items.forEach((item) => {
    if (!item?.name) return
    const key = item.name.replace('cache_default_', '')
    if (defaults[key]) {
      try {
        Object.assign(defaults[key], JSON.parse(item.value))
      } catch (e) {}
    }
  })
}

onMounted(() => {
  load()
})
</script>
