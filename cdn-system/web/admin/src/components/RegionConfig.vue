<template>
  <div class="region-config">
    <el-radio-group v-model="mode" class="mode-group">
      <el-radio value="disabled">不设置</el-radio>
      <el-radio value="foreign_exclude">国外 (不包括港澳台)</el-radio>
      <el-radio value="foreign_include">国外 (包括港澳台)</el-radio>
      <el-radio value="china_include">中国 (包括港澳台)</el-radio>
      <el-radio value="china_exclude">中国 (不包括港澳台)</el-radio>
      <el-radio value="custom">自定义</el-radio>
    </el-radio-group>

    <div v-if="mode === 'custom'" class="custom-regions">
      <div v-for="(region, index) in allRegions" :key="region.name" class="region-row">
        <div class="region-header">
          <div class="header-left">
            <span class="region-name">{{ region.name }}:</span>
            <el-checkbox
              v-model="region.checked"
              :indeterminate="region.indeterminate"
              @change="(val) => handleCheckAllChange(val, index)"
            >
              全选
            </el-checkbox>
            <el-button
              link
              type="primary"
              size="small"
              @click="region.expanded = !region.expanded"
              class="expand-btn"
            >
              {{ region.expanded ? '> 点击收起' : '> 点击展开' }}
            </el-button>
          </div>
        </div>
        
        <div v-show="region.expanded" class="region-content">
          <el-checkbox-group v-model="region.selected" @change="() => handleCheckedCountriesChange(index)">
            <el-checkbox v-for="country in region.countries" :key="country" :label="country" :value="country">
              {{ country }}
            </el-checkbox>
          </el-checkbox-group>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({ mode: 'disabled', countries: [] })
  }
})

const emit = defineEmits(['update:modelValue'])

const mode = ref('disabled')

// Data Source
const rawRegions = [
  {
    name: '亚洲',
    countries: ['蒙古', '朝鲜', '韩国', '日本', '越南', '老挝', '柬埔寨', '泰国', '缅甸', '马来西亚', '新加坡', '印度尼西亚', '文莱', '菲律宾', '东帝汶', '印度', '不丹', '尼泊尔', '孟加拉', '斯里兰卡', '马尔代夫', '巴基斯坦', '阿富汗', '伊朗', '伊拉克', '叙利亚', '约旦', '黎巴嫩', '以色列', '巴勒斯坦', '沙特阿拉伯', '科威特', '巴林', '卡塔尔', '阿联酋', '阿曼', '也门', '土库曼斯坦', '塔吉克斯坦', '吉尔吉斯', '乌兹别克', '哈萨克斯坦']
  },
  {
    name: '中国',
    countries: ['北京', '天津', '河北', '山西', '内蒙古', '辽宁', '吉林', '黑龙江', '上海', '江苏', '浙江', '安徽', '福建', '江西', '山东', '河南', '湖北', '湖南', '广东', '广西', '海南', '重庆', '四川', '贵州', '云南', '西藏', '陕西', '甘肃', '青海', '宁夏', '新疆', '香港', '澳门', '台湾']
  },
  {
    name: '北美洲',
    countries: ['美国', '加拿大', '墨西哥', '格陵兰', '危地马拉', '伯利兹', '萨尔瓦多', '洪都拉斯', '尼加拉瓜', '哥斯达黎加', '巴拿马', '其他北美地区']
  },
  {
    name: '南美洲',
    countries: ['巴西', '阿根廷', '智利', '哥伦比亚', '秘鲁', '委内瑞拉', '玻利维亚', '厄瓜多尔', '巴拉圭', '乌拉圭', '圭亚那', '苏里南', '法属圭亚那']
  },
  {
    name: '欧洲',
    countries: ['英国', '法国', '德国', '意大利', '西班牙', '葡萄牙', '荷兰', '比利时', '瑞士', '奥地利', '瑞典', '挪威', '芬兰', '丹麦', '冰岛', '爱尔兰', '波兰', '捷克', '斯洛伐克', '匈牙利', '罗马尼亚', '保加利亚', '希腊', '塞尔维亚', '克罗地亚', '波黑', '黑山', '北马其顿', '阿尔巴尼亚', '乌克兰', '白俄罗斯', '立陶宛', '拉脱维亚', '爱沙尼亚', '摩尔多瓦', '斯洛文尼亚', '卢森堡', '马耳他', '安道尔', '列支敦士登', '摩纳哥', '梵蒂冈', '俄罗斯']
  },
  {
    name: '大洋洲',
    countries: ['澳大利亚', '新西兰', '斐济', '巴布亚新几内亚', '所罗门群岛', '瓦努阿图', '萨摩亚', '汤加', '基里巴斯', '密克罗尼西亚', '马绍尔群岛', '帕劳', '瑙鲁', '图瓦卢']
  },
  {
      name: '非洲',
      countries: ['安哥拉', '贝宁', '博茨瓦纳', '布基纳法索', '布隆迪', '佛得角', '中非共和国', '乍得', '科摩罗', '刚果(布)', '刚果(金)', '吉布提', '埃及', '赤道几内亚', '厄立特里亚', '埃塞俄比亚', '加蓬', '冈比亚', '加纳', '几内亚', '几内亚比绍', '科特迪瓦', '肯尼亚', '莱索托', '利比里亚', '利比亚', '马达加斯加', '马拉维', '马里', '毛里塔尼亚', '毛里求斯', '摩洛哥', '莫桑比克', '纳米比亚', '尼日尔', '尼日利亚', '卢旺达', '圣多美和普林西比', '塞内加尔', '塞舌尔', '塞拉利昂', '索马里', '南非', '南苏丹', '苏丹', '斯威士兰', '坦桑尼亚', '多哥', '突尼斯', '乌干达', '赞比亚', '津巴布韦']
  }
]

// Extended state for UI
const allRegions = ref(rawRegions.map(r => ({
  ...r,
  checked: false,
  indeterminate: false,
  expanded: true, // Default expanded to specific request or all? Screenshot shows some expanded. Let's expand China by default or all. Screenshot shows Asia and China expanded.
  selected: []
})))

// Initialize from props
watch(() => props.modelValue, (val) => {
  if (val) {
    // Prevent infinite loop if internal update triggers this (though watch deep usually needed)
    if (val.mode !== mode.value) {
        mode.value = val.mode
    }
    
    if (val.countries && mode.value === 'custom') {
        const flatSelected = new Set(val.countries)
        allRegions.value.forEach(r => {
            r.selected = r.countries.filter(c => flatSelected.has(c))
            updateCheckState(r)
        })
    }
  }
}, { immediate: true, deep: true })

// Update parent
function emitUpdate() {
    const countries = []
    if (mode.value === 'custom') {
        allRegions.value.forEach(r => {
            countries.push(...r.selected)
        })
    }
    emit('update:modelValue', {
        mode: mode.value,
        countries: countries
    })
}

watch(mode, () => {
    // If switching to custom, maybe preserve previous selection? 
    // Currently we just emit.
    emitUpdate()
})

function handleCheckAllChange(val, index) {
  const region = allRegions.value[index]
  region.selected = val ? [...region.countries] : []
  region.indeterminate = false
  emitUpdate()
}

function handleCheckedCountriesChange(index) {
  const region = allRegions.value[index]
  updateCheckState(region)
  emitUpdate()
}

function updateCheckState(region) {
    const checkedCount = region.selected.length
    const totalCount = region.countries.length
    region.checked = checkedCount === totalCount && totalCount > 0
    region.indeterminate = checkedCount > 0 && checkedCount < totalCount
}

</script>

<style scoped>
.region-config {
  width: 100%;
}
.mode-group {
  margin-bottom: 20px;
}
.region-row {
  margin-bottom: 15px;
}
.region-header {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.region-name {
  font-weight: bold;
  min-width: 60px;
  color: #606266;
  font-size: 14px;
}
.expand-btn {
  font-size: 12px;
}
.region-content {
  padding-left: 72px; /* Indent to align with content */
}
:deep(.el-checkbox) {
    margin-right: 20px;
    margin-bottom: 8px;
    width: 100px; /* Fixed width for grid-like alignment as in screenshot */
}
:deep(.el-checkbox-group) {
    display: flex;
    flex-wrap: wrap;
}
</style>
