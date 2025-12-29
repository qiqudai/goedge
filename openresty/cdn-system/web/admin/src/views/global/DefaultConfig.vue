<template>
  <div class="default-config">
    <el-card shadow="never" class="layout-card">
      <template #header>
        <div class="card-header">全局默认配置</div>
      </template>

      <el-tabs v-model="topTab" class="top-tabs">
        <el-tab-pane label="全局配置" name="global">
          <el-tabs v-model="globalTab" class="sub-tabs">
            <el-tab-pane label="网站" name="site">
              <site-config />
            </el-tab-pane>
            <el-tab-pane label="转发" name="stream">
              <stream-config />
            </el-tab-pane>
            <el-tab-pane label="证书" name="cert">
              <cert-config />
            </el-tab-pane>
          </el-tabs>
        </el-tab-pane>

        <el-tab-pane label="缓存配置" name="cache">
          <cache-config />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import SiteConfig from './components/SiteConfig.vue'
import StreamConfig from './components/StreamConfig.vue'
import CertConfig from './components/CertConfig.vue'
import CacheConfig from './components/CacheConfig.vue'

const topTab = ref('global')
const globalTab = ref('site')
</script>

<style scoped>
@import "./DefaultConfig.css";
</style>
