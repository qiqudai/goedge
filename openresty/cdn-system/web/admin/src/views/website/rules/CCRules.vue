<template>
  <el-tabs v-model="activeTab">
    <el-tab-pane label="规则组" name="groups">
      <GroupList />
    </el-tab-pane>
    <el-tab-pane label="匹配器" name="matchers">
      <MatcherList />
    </el-tab-pane>
    <el-tab-pane label="过滤器" name="filters">
      <FilterList />
    </el-tab-pane>
  </el-tabs>
</template>

<script setup>
import { ref } from 'vue'
import GroupList from './cc/GroupList.vue'
import MatcherList from './cc/MatcherList.vue'
import FilterList from './cc/FilterList.vue'

const activeTab = ref('groups')
</script>
