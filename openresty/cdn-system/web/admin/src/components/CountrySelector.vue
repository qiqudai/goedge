<template>
  <div class="country-selector">
    <el-checkbox-group v-model="innerValue">
      <div v-for="group in groups" :key="group.label" class="country-group">
        <div class="group-title">{{ group.label }}</div>
        <div class="group-items">
          <el-checkbox v-for="item in group.items" :key="item" :label="item">
            {{ item }}
          </el-checkbox>
        </div>
      </div>
    </el-checkbox-group>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue'])

const innerValue = computed({
  get() {
    return props.modelValue
  },
  set(value) {
    emit('update:modelValue', value)
  }
})

const groups = [
  {
    label: '亚洲',
    items: ['中国境内', '香港', '澳门', '台湾', '蒙古', '朝鲜', '韩国', '日本', '越南', '老挝', '柬埔寨', '泰国', '缅甸', '马来西亚', '新加坡', '印度尼西亚', '文莱', '菲律宾', '东帝汶', '印度', '不丹', '尼泊尔', '孟加拉', '斯里兰卡', '马尔代夫', '巴基斯坦', '阿富汗', '伊朗', '伊拉克', '叙利亚', '约旦', '黎巴嫩', '以色列', '巴勒斯坦', '沙特阿拉伯', '科威特', '巴林', '卡塔尔', '阿联酋', '阿曼', '也门', '土库曼斯坦', '塔吉克斯坦', '吉尔吉斯', '乌兹别克', '哈萨克斯坦']
  },
  {
    label: '非洲',
    items: ['安哥拉', '贝宁', '博茨瓦纳', '布基纳法索', '布隆迪', '佛得角', '中非共和国', '乍得', '科摩罗', '刚果(布)', '刚果(金)', '吉布提', '埃及', '赤道几内亚', '厄立特里亚', '埃塞俄比亚', '加蓬', '冈比亚', '加纳', '几内亚', '几内亚比绍', '科特迪瓦', '肯尼亚', '莱索托', '利比里亚', '利比亚', '马达加斯加', '马拉维', '马里', '毛里塔尼亚', '毛里求斯', '摩洛哥', '莫桑比克', '纳米比亚', '尼日尔', '尼日利亚', '卢旺达', '圣多美和普林西比', '塞内加尔', '塞舌尔', '塞拉利昂', '索马里', '南非', '南苏丹', '苏丹', '斯威士兰', '坦桑尼亚', '多哥', '突尼斯', '乌干达', '赞比亚', '津巴布韦']
  },
  {
    label: '欧洲',
    items: ['英国', '法国', '德国', '意大利', '西班牙', '葡萄牙', '荷兰', '比利时', '瑞士', '奥地利', '瑞典', '挪威', '芬兰', '丹麦', '冰岛', '爱尔兰', '波兰', '捷克', '斯洛伐克', '匈牙利', '罗马尼亚', '保加利亚', '希腊', '塞尔维亚', '克罗地亚', '波黑', '黑山', '北马其顿', '阿尔巴尼亚', '乌克兰', '白俄罗斯', '立陶宛', '拉脱维亚', '爱沙尼亚', '摩尔多瓦', '斯洛文尼亚', '卢森堡', '马耳他', '安道尔', '列支敦士登', '摩纳哥', '梵蒂冈']
  },
  {
    label: '北美洲',
    items: ['美国', '加拿大', '墨西哥', '格陵兰', '危地马拉', '伯利兹', '萨尔瓦多', '洪都拉斯', '尼加拉瓜', '哥斯达黎加', '巴拿马']
  },
  {
    label: '南美洲',
    items: ['巴西', '阿根廷', '智利', '哥伦比亚', '秘鲁', '委内瑞拉', '玻利维亚', '厄瓜多尔', '巴拉圭', '乌拉圭', '圭亚那', '苏里南']
  },
  {
    label: '大洋洲',
    items: ['澳大利亚', '新西兰', '斐济', '巴布亚新几内亚', '所罗门群岛', '瓦努阿图', '萨摩亚', '汤加', '基里巴斯', '密克罗尼西亚', '马绍尔群岛', '帕劳', '瑙鲁', '图瓦卢']
  }
]
</script>

<style scoped>
.country-selector {
  border: 1px dashed #dcdfe6;
  padding: 12px;
  border-radius: 6px;
  margin-top: 8px;
}
.country-group {
  margin-bottom: 12px;
}
.group-title {
  font-size: 13px;
  color: #606266;
  margin-bottom: 6px;
}
.group-items {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
}
</style>
