<template>
  <div class="common-layout">
    <el-container>
      <el-aside width="200px" class="aside">
        <div class="logo">CDN Admin</div>
        <el-menu
          :default-active="activeMenu"
          class="el-menu-vertical-demo"
          router
          :background-color="menuBackground"
          :text-color="menuTextColor"
          :active-text-color="menuActiveColor"
        >
          <template v-for="routeItem in displayedRoutes" :key="routeItem.path">
            <el-sub-menu
              v-if="routeItem.children && routeItem.children.length > 0 && !routeItem.meta?.hidden"
              :index="resolvePath(routeItem)"
            >
              <template #title>
                <el-icon v-if="routeItem.meta && routeItem.meta.icon">
                  <component :is="getIcon(routeItem.meta.icon)" />
                </el-icon>
                <span>{{ routeItem.meta?.title || routeItem.name }}</span>
              </template>
              <template v-for="child in routeItem.children" :key="child.path">
                <el-menu-item v-if="!child.meta?.hidden" :index="resolvePath(routeItem, child)">
                  <el-icon v-if="child.meta && child.meta.icon">
                    <component :is="getIcon(child.meta.icon)" />
                  </el-icon>
                  <span>{{ child.meta?.title || child.name }}</span>
                </el-menu-item>
              </template>
            </el-sub-menu>

            <el-menu-item v-else :index="resolvePath(routeItem)">
              <el-icon v-if="routeItem.meta && routeItem.meta.icon">
                <component :is="getIcon(routeItem.meta.icon)" />
              </el-icon>
              <span>{{ routeItem.meta?.title || routeItem.name }}</span>
            </el-menu-item>
          </template>
        </el-menu>
      </el-aside>
      <el-container>
        <el-header class="header">
          <div class="header-content">
            <div class="header-title">管理后台</div>
            <div class="header-actions">
              <div class="theme-toggle">
                <el-icon class="theme-icon" :class="{ active: !isDark }"><Sunny /></el-icon>
                <el-switch v-model="isDark" @change="toggleTheme" />
                <el-icon class="theme-icon" :class="{ active: isDark }"><Moon /></el-icon>
              </div>
              <el-dropdown>
                <span class="user-trigger">
                  <el-icon><User /></el-icon>
                </span>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item v-if="isAdmin" @click="goSystemSettings">系统设置</el-dropdown-item>
                    <el-dropdown-item v-else @click="goProfile">个人中心</el-dropdown-item>
                    <el-dropdown-item divided @click="logout">退出登录</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </el-header>
        <el-main>
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  User,
  DataLine,
  Connection,
  Setting,
  Document,
  DocumentCopy,
  Upload,
  Money,
  FullScreen,
  Monitor,
  Sunny,
  Moon
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()

const role = localStorage.getItem('role') || 'user'
const isAdmin = computed(() => role === 'admin')

const isDark = ref(false)

const menuBackground = 'var(--sidebar-bg)'
const menuTextColor = 'var(--sidebar-text)'
const menuActiveColor = 'var(--sidebar-active)'

const applyTheme = () => {
  const theme = isDark.value ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', theme)
  localStorage.setItem('theme', theme)
}

const toggleTheme = () => {
  applyTheme()
}

const goSystemSettings = () => {
  router.push('/system/config')
}

const goProfile = () => {
  router.push('/account/profile')
}

const logout = () => {
  localStorage.removeItem('admin_token')
  router.push('/login')
}

const iconMap = {
  user: User,
  'data-line': DataLine,
  connection: Connection,
  setting: Setting,
  document: Document,
  'document-copy': DocumentCopy,
  upload: Upload,
  money: Money,
  'full-screen': FullScreen,
  dashboard: FullScreen,
  monitor: Monitor
}

const getIcon = name => iconMap[name] || DataLine

const hasPermission = targetRoute => {
  if (targetRoute.meta && targetRoute.meta.roles) {
    return targetRoute.meta.roles.includes(role)
  }
  return true
}

const filterRoutes = routes => {
  const res = []
  routes.forEach(routeItem => {
    const tmp = { ...routeItem }
    if (hasPermission(tmp)) {
      if (tmp.children) {
        tmp.children = filterRoutes(tmp.children)
      }
      res.push(tmp)
    }
  })
  return res
}

const displayedRoutes = computed(() => {
  const mainRoute = router.options.routes.find(r => r.path === '/')
  if (mainRoute && mainRoute.children) {
    const visible = mainRoute.children.filter(child => !child.meta?.hidden)
    return filterRoutes(visible)
  }
  return []
})

const activeMenu = computed(() => route.path)

const resolvePath = (routeItem, childItem) => {
  const parentPath = routeItem.path.replace(/^\/+|\/+$/g, '')

  if (childItem) {
    const childPath = childItem.path.replace(/^\/+|\/+$/g, '')
    return `/${parentPath}/${childPath}`
  }

  if (!parentPath) return '/'
  return '/' + parentPath
}

onMounted(() => {
  const savedTheme = localStorage.getItem('theme') || 'light'
  isDark.value = savedTheme === 'dark'
  applyTheme()
})
</script>

<style scoped>
.common-layout {
  height: 100vh;
  background-color: var(--content-bg);
  color: var(--text-color);
}
.el-container {
  height: 100%;
}
.aside {
  background-color: var(--sidebar-bg);
  color: var(--sidebar-text);
}
.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  font-weight: bold;
  font-size: 20px;
  background-color: var(--sidebar-logo-bg);
}
.header {
  background-color: var(--header-bg);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  color: var(--text-color);
}
.header-content {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.header-title {
  font-weight: 600;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}
.theme-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
}
.theme-icon {
  color: var(--muted-text);
}
.theme-icon.active {
  color: var(--text-color);
}
.user-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  cursor: pointer;
}
</style>
