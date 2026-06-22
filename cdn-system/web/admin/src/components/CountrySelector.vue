<template>
  <div class="country-selector">
    <!-- 顶部单选框组 -->
    <div class="mode-selector">
      <el-radio-group v-model="mode" @change="handleModeChange">
        <el-radio value="none">不设置</el-radio>
        <el-radio value="foreign_exclude">国外（不包括港澳台）</el-radio>
        <el-radio value="foreign_include">国外（包括港澳台）</el-radio>
        <el-radio value="china_include">中国（包括港澳台）</el-radio>
        <el-radio value="china_exclude">中国（不包括港澳台）</el-radio>
        <el-radio value="custom">自定义</el-radio>
      </el-radio-group>
    </div>

    <!-- 自定义部分 -->
    <div v-if="mode === 'custom'" class="custom-selector">

        <div v-for="(group, index) in displayGroups" :key="group.label" class="country-group">
          <div class="group-header">
            <div class="header-left">
              <span class="group-label">{{ group.label }}：</span>
              <el-checkbox
                :model-value="isGroupAllChecked(group)"
                :indeterminate="isGroupIndeterminate(group)"
                @change="(val) => handleGroupAllCheck(group, val)"
              >
                全选
              </el-checkbox>
              <el-button
                link
                type="primary"
                size="small"
                @click="toggleExpand(index)"
                class="expand-btn"
              >
                {{ group.expanded ? '收起' : '展开' }}
              </el-button>
            </div>
          </div>
          <el-checkbox-group v-show="group.expanded" v-model="innerValue" class="group-items">
            <el-checkbox
              v-for="item in group.items"
              :key="item.value"
              :label="item.value"
              :value="item.value"
            >
              {{ item.label }}
            </el-checkbox>
          </el-checkbox-group>
        </div>

    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

const emitUpdate = (value) => {
  emit('update:modelValue', value)
  emit('change', value)
}

const innerValue = computed({
  get() {
    return props.modelValue || []
  },
  set(value) {
    emitUpdate(value)
  }
})

const mode = ref('none')

// Define groups with Label and Value (Code)
const rawGroups = [
  {
    label: '亚洲',
    items: [
      { label: '中国境内', value: 'cn' },
      { label: '香港', value: 'hk' },
      { label: '澳门', value: 'mo' },
      { label: '台湾', value: 'tw' },
      { label: '蒙古', value: 'mn' },
      { label: '朝鲜', value: 'kp' },
      { label: '韩国', value: 'kr' },
      { label: '日本', value: 'jp' },
      { label: '越南', value: 'vn' },
      { label: '老挝', value: 'la' },
      { label: '柬埔寨', value: 'kh' },
      { label: '泰国', value: 'th' },
      { label: '缅甸', value: 'mm' },
      { label: '马来西亚', value: 'my' },
      { label: '新加坡', value: 'sg' },
      { label: '印度尼西亚', value: 'id' },
      { label: '文莱', value: 'bn' },
      { label: '菲律宾', value: 'ph' },
      { label: '东帝汶', value: 'tl' },
      { label: '印度', value: 'in' },
      { label: '不丹', value: 'bt' },
      { label: '尼泊尔', value: 'np' },
      { label: '孟加拉', value: 'bd' },
      { label: '斯里兰卡', value: 'lk' },
      { label: '马尔代夫', value: 'mv' },
      { label: '巴基斯坦', value: 'pk' },
      { label: '阿富汗', value: 'af' },
      { label: '伊朗', value: 'ir' },
      { label: '伊拉克', value: 'iq' },
      { label: '叙利亚', value: 'sy' },
      { label: '约旦', value: 'jo' },
      { label: '黎巴嫩', value: 'lb' },
      { label: '以色列', value: 'il' },
      { label: '巴勒斯坦', value: 'ps' },
      { label: '沙特阿拉伯', value: 'sa' },
      { label: '科威特', value: 'kw' },
      { label: '巴林', value: 'bh' },
      { label: '卡塔尔', value: 'qa' },
      { label: '阿联酋', value: 'ae' },
      { label: '阿曼', value: 'om' },
      { label: '也门', value: 'ye' },
      { label: '土库曼斯坦', value: 'tm' },
      { label: '塔吉克斯坦', value: 'tj' },
      { label: '吉尔吉斯', value: 'kg' },
      { label: '乌兹别克', value: 'uz' },
      { label: '哈萨克斯坦', value: 'kz' },
      { label: '亚美尼亚', value: 'am' },
      { label: '阿塞拜疆', value: 'az' },
      { label: '格鲁吉亚', value: 'ge' },
      { label: '土耳其', value: 'tr' }
    ]
  },
  {
    label: '非洲',
    items: [
      { label: '阿尔及利亚', value: 'dz' },
      { label: '安哥拉', value: 'ao' },
      { label: '贝宁', value: 'bj' },
      { label: '博茨瓦纳', value: 'bw' },
      { label: '布基纳法索', value: 'bf' },
      { label: '布隆迪', value: 'bi' },
      { label: '佛得角', value: 'cv' },
      { label: '中非共和国', value: 'cf' },
      { label: '乍得', value: 'td' },
      { label: '科摩罗', value: 'km' },
      { label: '刚果(布)', value: 'cg' },
      { label: '刚果(金)', value: 'cd' },
      { label: '吉布提', value: 'dj' },
      { label: '埃及', value: 'eg' },
      { label: '赤道几内亚', value: 'gq' },
      { label: '厄立特里亚', value: 'er' },
      { label: '埃塞俄比亚', value: 'et' },
      { label: '加蓬', value: 'ga' },
      { label: '喀麦隆', value: 'cm' },
      { label: '冈比亚', value: 'gm' },
      { label: '加纳', value: 'gh' },
      { label: '几内亚', value: 'gn' },
      { label: '几内亚比绍', value: 'gw' },
      { label: '科特迪瓦', value: 'ci' },
      { label: '肯尼亚', value: 'ke' },
      { label: '莱索托', value: 'ls' },
      { label: '利比里亚', value: 'lr' },
      { label: '利比亚', value: 'ly' },
      { label: '马达加斯加', value: 'mg' },
      { label: '马拉维', value: 'mw' },
      { label: '马里', value: 'ml' },
      { label: '毛里塔尼亚', value: 'mr' },
      { label: '毛里求斯', value: 'mu' },
      { label: '摩洛哥', value: 'ma' },
      { label: '莫桑比克', value: 'mz' },
      { label: '纳米比亚', value: 'na' },
      { label: '尼日尔', value: 'ne' },
      { label: '尼日利亚', value: 'ng' },
      { label: '卢旺达', value: 'rw' },
      { label: '圣多美和普林西比', value: 'st' },
      { label: '塞内加尔', value: 'sn' },
      { label: '塞舌尔', value: 'sc' },
      { label: '塞拉利昂', value: 'sl' },
      { label: '索马里', value: 'so' },
      { label: '南非', value: 'za' },
      { label: '南苏丹', value: 'ss' },
      { label: '苏丹', value: 'sd' },
      { label: '斯威士兰', value: 'sz' },
      { label: '坦桑尼亚', value: 'tz' },
      { label: '多哥', value: 'tg' },
      { label: '突尼斯', value: 'tn' },
      { label: '乌干达', value: 'ug' },
      { label: '赞比亚', value: 'zm' },
      { label: '津巴布韦', value: 'zw' }
    ]
  },
  {
    label: '欧洲',
    items: [
      { label: '英国', value: 'gb' },
      { label: '法国', value: 'fr' },
      { label: '德国', value: 'de' },
      { label: '意大利', value: 'it' },
      { label: '西班牙', value: 'es' },
      { label: '葡萄牙', value: 'pt' },
      { label: '荷兰', value: 'nl' },
      { label: '比利时', value: 'be' },
      { label: '瑞士', value: 'ch' },
      { label: '奥地利', value: 'at' },
      { label: '瑞典', value: 'se' },
      { label: '挪威', value: 'no' },
      { label: '芬兰', value: 'fi' },
      { label: '丹麦', value: 'dk' },
      { label: '冰岛', value: 'is' },
      { label: '爱尔兰', value: 'ie' },
      { label: '波兰', value: 'pl' },
      { label: '捷克', value: 'cz' },
      { label: '斯洛伐克', value: 'sk' },
      { label: '匈牙利', value: 'hu' },
      { label: '罗马尼亚', value: 'ro' },
      { label: '俄罗斯', value: 'ru' },
      { label: '保加利亚', value: 'bg' },
      { label: '希腊', value: 'gr' },
      { label: '塞尔维亚', value: 'rs' },
      { label: '圣马力诺', value: 'sm' },
      { label: '克罗地亚', value: 'hr' },
      { label: '波黑', value: 'ba' },
      { label: '黑山', value: 'me' },
      { label: '北马其顿', value: 'mk' },
      { label: '阿尔巴尼亚', value: 'al' },
      { label: '乌克兰', value: 'ua' },
      { label: '白俄罗斯', value: 'by' },
      { label: '立陶宛', value: 'lt' },
      { label: '拉脱维亚', value: 'lv' },
      { label: '爱沙尼亚', value: 'ee' },
      { label: '摩尔多瓦', value: 'md' },
      { label: '塞浦路斯', value: 'cy' },
      { label: '斯洛文尼亚', value: 'si' },
      { label: '卢森堡', value: 'lu' },
      { label: '马耳他', value: 'mt' },
      { label: '安道尔', value: 'ad' },
      { label: '列支敦士登', value: 'li' },
      { label: '摩纳哥', value: 'mc' },
      { label: '梵蒂冈', value: 'va' }
    ]
  },
  {
    label: '北美洲',
    items: [
      { label: '美国', value: 'us' },
      { label: '加拿大', value: 'ca' },
      { label: '墨西哥', value: 'mx' },
      { label: '格陵兰', value: 'gl' },
      { label: '危地马拉', value: 'gt' },
      { label: '伯利兹', value: 'bz' },
      { label: '萨尔瓦多', value: 'sv' },
      { label: '洪都拉斯', value: 'hn' },
      { label: '尼加拉瓜', value: 'ni' },
      { label: '哥斯达黎加', value: 'cr' },
      { label: '巴拿马', value: 'pa' },
      { label: '古巴', value: 'cu' },
      { label: '海地', value: 'ht' },
      { label: '多米尼加', value: 'do' },
      { label: '牙买加', value: 'jm' },
      { label: '巴哈马', value: 'bs' },
      { label: '巴巴多斯', value: 'bb' },
      { label: '特立尼达和多巴哥', value: 'tt' },
      { label: '安提瓜和巴布达', value: 'ag' },
      { label: '多米尼克', value: 'dm' },
      { label: '格林纳达', value: 'gd' },
      { label: '圣基茨和尼维斯', value: 'kn' },
      { label: '圣卢西亚', value: 'lc' },
      { label: '圣文森特和格林纳丁斯', value: 'vc' },
      { label: '阿鲁巴', value: 'aw' }
    ]
  },
  {
    label: '南美洲',
    items: [
      { label: '巴西', value: 'br' },
      { label: '阿根廷', value: 'ar' },
      { label: '智利', value: 'cl' },
      { label: '哥伦比亚', value: 'co' },
      { label: '秘鲁', value: 'pe' },
      { label: '委内瑞拉', value: 've' },
      { label: '玻利维亚', value: 'bo' },
      { label: '厄瓜多尔', value: 'ec' },
      { label: '巴拉圭', value: 'py' },
      { label: '乌拉圭', value: 'uy' },
      { label: '圭亚那', value: 'gy' },
      { label: '苏里南', value: 'sr' },
      { label: '法属圭亚那', value: 'gf' }
    ]
  },
  {
    label: '大洋洲',
    items: [
      { label: '澳大利亚', value: 'au' },
      { label: '新西兰', value: 'nz' },
      { label: '斐济', value: 'fj' },
      { label: '巴布亚新几内亚', value: 'pg' },
      { label: '所罗门群岛', value: 'sb' },
      { label: '瓦努阿图', value: 'vu' },
      { label: '萨摩亚', value: 'ws' },
      { label: '汤加', value: 'to' },
      { label: '基里巴斯', value: 'ki' },
      { label: '密克罗尼西亚', value: 'fm' },
      { label: '马绍尔群岛', value: 'mh' },
      { label: '帕劳', value: 'pw' },
      { label: '瑙鲁', value: 'nr' },
      { label: '图瓦卢', value: 'tv' }
    ]
  }
]

// Reactive groups state for UI (expanded/collapsed)
const displayGroups = ref(rawGroups.map(g => ({
  ...g,
  expanded: true
})))

const toggleExpand = (index) => {
  displayGroups.value[index].expanded = !displayGroups.value[index].expanded
}

// Constants for Presets (using CODES)
const CHINA_INLAND_CODE = 'cn'
const CN_REGIONS_CODES = ['hk', 'mo', 'tw']
const ALL_ITEMS_CODES = rawGroups.flatMap(g => g.items.map(i => i.value))

// Cache comparison arrays to avoid recreation
const PRESET_CHINA_EXCLUDE = [CHINA_INLAND_CODE]
const PRESET_CHINA_INCLUDE = [CHINA_INLAND_CODE, ...CN_REGIONS_CODES]
const PRESET_FOREIGN_INCLUDE = ALL_ITEMS_CODES.filter(i => i !== CHINA_INLAND_CODE)
const PRESET_FOREIGN_EXCLUDE = ALL_ITEMS_CODES.filter(i => i !== CHINA_INLAND_CODE && !CN_REGIONS_CODES.includes(i))

// Helpers
const areSetsEqual = (arr1, arr2) => {
  // Safe check if standard array or proxy
  const a1 = arr1 || []
  const a2 = arr2 || []
  if (a1.length !== a2.length) return false
  const s2 = new Set(a2)
  return a1.every(x => s2.has(x))
}

const detectMode = (val) => {
  if (!val || val.length === 0) return 'none'

  if (areSetsEqual(val, PRESET_CHINA_EXCLUDE)) return 'china_exclude'
  if (areSetsEqual(val, PRESET_CHINA_INCLUDE)) return 'china_include'
  if (areSetsEqual(val, PRESET_FOREIGN_INCLUDE)) return 'foreign_include'
  if (areSetsEqual(val, PRESET_FOREIGN_EXCLUDE)) return 'foreign_exclude'

  return 'custom'
}

const handleModeChange = (newMode) => {
  let newValue = []
  switch (newMode) {
    case 'none':
      newValue = []
      break
    case 'china_exclude':
      newValue = [...PRESET_CHINA_EXCLUDE]
      break
    case 'china_include':
      newValue = [...PRESET_CHINA_INCLUDE]
      break
    case 'foreign_include':
      newValue = [...PRESET_FOREIGN_INCLUDE]
      break
    case 'foreign_exclude':
      newValue = [...PRESET_FOREIGN_EXCLUDE]
      break
    case 'custom':
      // Do nothing to value, just switch mode
      return
  }
  emitUpdate(newValue)
}

const isGroupAllChecked = (group) => {
  if (!props.modelValue || group.items.length === 0) return false
  const currentSet = new Set(props.modelValue)
  // Check if ALL item VALUES in group are in existing modelValue
  return group.items.every(item => currentSet.has(item.value))
}

const isGroupIndeterminate = (group) => {
  if (!props.modelValue || props.modelValue.length === 0) return false
  const currentSet = new Set(props.modelValue)
  let count = 0
  for (const item of group.items) {
    if (currentSet.has(item.value)) count++
  }
  return count > 0 && count < group.items.length
}

const handleGroupAllCheck = (group, checked) => {
  const currentSet = new Set(props.modelValue || [])
  
  if (checked) {
    group.items.forEach(item => currentSet.add(item.value))
  } else {
    group.items.forEach(item => currentSet.delete(item.value))
  }
  
  emitUpdate(Array.from(currentSet))
}

watch(() => props.modelValue, (newVal) => {
  const detected = detectMode(newVal)
  if (mode.value !== 'custom') {
    if (detected !== mode.value) {
      mode.value = detected
    }
  } else {
    // If we are in custom mode, we track if user MANUALLY matches a preset
    if (detected !== 'custom' && detected !== 'none') {
        mode.value = detected
    }
  }
}, { deep: true })

onMounted(() => {
  mode.value = detectMode(props.modelValue)
})
</script>

<style scoped>
.country-selector {
  border: 1px dashed #dcdfe6;
  padding: 12px;
  border-radius: 6px;
  margin-top: 8px;
}
.mode-selector {
  margin-bottom: 20px;
}
.custom-selector {
  border-top: 1px dashed #eee;
  padding-top: 12px;
}
.country-group {
  margin-bottom: 16px;
}
.group-header {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}
.header-left {
  display: flex;
  align-items: center;
}
.group-label {
  font-size: 14px;
  font-weight: bold;
  color: #606266;
  margin-right: 12px;
  min-width: 60px;
}
.expand-btn {
  margin-left: 12px;
  font-size: 12px;
}
.group-items {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
  padding-left: 0;
}
</style>
