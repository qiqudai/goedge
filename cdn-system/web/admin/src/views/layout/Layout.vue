<template>
  <div class="common-layout">
    <el-container>
      <el-aside width="200px" class="aside">
        <div class="logo">
          <img v-if="logoUrl" :src="logoUrl" class="logo-img" alt="logo" />
          <span v-else>{{ sidebarTitle }}</span>
        </div>
        <el-menu
          :default-active="activeMenu"
          class="el-menu-vertical-demo"
          router
          :unique-opened="true"
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
            <div class="header-title">{{ consoleTitle }}</div>
            <div class="header-actions">
              <div v-if="!isAdmin" class="message-badge" @click="goMessages">
                <el-icon class="message-icon"><Bell /></el-icon>
                <span v-if="unreadCount > 0" class="message-count">{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
              </div>
              <div class="theme-toggle">
                <button class="theme-switch" :class="{ dark: isDark }" type="button" @click="toggleTheme">
                  <span class="theme-thumb">
                    <el-icon><component :is="isDark ? Moon : Sunny" /></el-icon>
                  </span>
                </button>
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
        <el-footer v-if="showFooter" class="footer">
          <div v-if="footerLinks.length" class="footer-links">
            <a
              v-for="item in footerLinks"
              :key="item.label"
              :href="item.url"
              target="_blank"
              rel="noopener"
            >{{ item.label }}</a>
          </div>
          <div v-if="footerCopy" class="footer-copy">{{ footerCopy }}</div>
        </el-footer>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount, watch } from 'vue'
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
  Moon,
  Bell
} from '@element-plus/icons-vue'
import { ElNotification } from 'element-plus'
import request from '@/utils/request'
import { useSystemInfo } from '@/composables/useSystemInfo'

const router = useRouter()
const route = useRoute()

const role = localStorage.getItem('role') || 'user'
const isAdmin = computed(() => role === 'admin')
const { systemInfo, loadSystemInfo } = useSystemInfo()

const isDark = ref(false)
const unreadCount = ref(0)
const lastMessageId = ref(Number(localStorage.getItem('last_message_id') || 0))
let messageTimer = null

const menuBackground = 'var(--sidebar-bg)'
const menuTextColor = 'var(--sidebar-text)'
const menuActiveColor = 'var(--sidebar-active)'

const logoUrl = computed(() => systemInfo.logo_file || '')
const sidebarTitle = computed(() => systemInfo.sys_name || 'CDN Admin')
const consoleTitle = computed(() => {
  if (isAdmin.value) {
    return systemInfo.admin_console_title || systemInfo.sys_name || '管理后台'
  }
  return systemInfo.user_console_title || systemInfo.sys_name || '控制台'
})
const footerCopy = computed(() => systemInfo.footer_copyright || '')
const footerLinks = computed(() => {
  const raw = systemInfo.footer_link || ''
  if (!raw) return []
  return raw
    .split(/\r?\n/)
    .map(line => line.trim())
    .filter(Boolean)
    .map(line => {
      const [label, url] = line.split('|').map(part => part.trim())
      return { label: label || url, url }
    })
    .filter(item => item.url)
})
const showFooter = computed(() => footerLinks.value.length > 0 || !!footerCopy.value)

const applyTheme = () => {
  const theme = isDark.value ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', theme)
  localStorage.setItem('theme', theme)
}

const toggleTheme = () => {
  isDark.value = !isDark.value
  applyTheme()
}

const goSystemSettings = () => {
  router.push('/system/config')
}

const goProfile = () => {
  router.push('/account/profile')
}

const goMessages = () => {
  router.push('/account/messages')
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

const pollUnread = async () => {
  if (isAdmin.value) {
    return
  }
  try {
    const res = await request.get('/messages/unread', { skipLoading: true })
    const data = res.data || {}
    const count = data.count || 0
    unreadCount.value = count
    const latest = data.latest || {}
    if (latest.id && latest.id !== lastMessageId.value) {
      lastMessageId.value = latest.id
      localStorage.setItem('last_message_id', String(latest.id))
      ElNotification({
        title: '新消息',
        message: latest.title || '收到新通知',
        type: 'info',
        duration: 4500,
        onClick: goMessages
      })
    }
  } catch (e) {
    // ignore polling errors
  }
}

const startPolling = () => {
  if (messageTimer) {
    return
  }
  messageTimer = setInterval(pollUnread, 30000)
}

const stopPolling = () => {
  if (messageTimer) {
    clearInterval(messageTimer)
    messageTimer = null
  }
}

onMounted(() => {
  const savedTheme = localStorage.getItem('theme') || 'light'
  isDark.value = savedTheme === 'dark'
  applyTheme()
  loadSystemInfo()
  if (!isAdmin.value) {
    pollUnread()
    startPolling()
  }
})

onBeforeUnmount(() => {
  stopPolling()
})

watch(consoleTitle, (val) => {
  if (val) {
    document.title = val
  }
}, { immediate: true })
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
.logo-img {
  max-width: 140px;
  max-height: 40px;
  margin-top: 10px;
  object-fit: contain;
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
.message-badge {
  cursor: pointer;
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.message-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
}
.message-count {
  position: absolute;
  top: -4px;
  right: -6px;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: 999px;
  background: #f56c6c;
  color: #fff;
  font-size: 11px;
  line-height: 18px;
  text-align: center;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
}
.theme-toggle {
  display: flex;
  align-items: center;
}
.theme-switch {
  position: relative;
  width: 56px;
  height: 28px;
  border-radius: 999px;
  border: 1px solid var(--border-color);
  background: linear-gradient(135deg, #fef3c7, #dbeafe);
  padding: 0;
  cursor: pointer;
  transition: background 0.3s ease;
}
.theme-switch.dark {
  background: linear-gradient(135deg, #0f172a, #1f2937);
}
.theme-thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #f59e0b;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
  transition: transform 0.28s ease, color 0.28s ease;
}
.theme-switch.dark .theme-thumb {
  transform: translateX(28px);
  color: #111827;
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
.footer {
  background: var(--header-bg);
  border-top: 1px solid var(--border-color);
  color: var(--muted-text);
  font-size: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 16px;
  gap: 12px;
}
.footer-links {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.footer-links a {
  color: inherit;
  text-decoration: none;
}
.footer-links a:hover {
  color: var(--text-color);
}
.footer-copy {
  white-space: nowrap;
}
</style>
