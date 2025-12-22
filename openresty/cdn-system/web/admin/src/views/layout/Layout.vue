<template>
  <div class="common-layout">
    <el-container>
      <el-aside width="200px" class="aside">
        <div class="logo">CDN Admin</div>
        <el-menu
          :default-active="activeMenu"
          class="el-menu-vertical-demo"
          router
          background-color="#545c64"
          text-color="#fff"
          active-text-color="#ffd04b">
          
          <template v-for="route in displayedRoutes" :key="route.path">
             <!-- Submenu -->
             <el-sub-menu v-if="route.children && route.children.length > 0 && !route.meta?.hidden" :index="resolvePath(route)">
                <template #title>
                    <el-icon v-if="route.meta && route.meta.icon">
                        <component :is="getIcon(route.meta.icon)" />
                    </el-icon>
                    <span>{{ route.meta?.title || route.name }}</span>
                </template>
                <template v-for="child in route.children" :key="child.path">
                    <el-menu-item  v-if="!child.meta?.hidden" :index="resolvePath(route, child)">
                        <el-icon v-if="child.meta && child.meta.icon">
                            <component :is="getIcon(child.meta.icon)" />
                        </el-icon>
                        <span>{{ child.meta?.title || child.name }}</span>
                    </el-menu-item>
                </template>
             </el-sub-menu>

             <!-- Leaf Item -->
             <el-menu-item v-else :index="resolvePath(route)">
                <el-icon v-if="route.meta && route.meta.icon">
                    <component :is="getIcon(route.meta.icon)" />
                </el-icon>
                <span>{{ route.meta?.title || route.name }}</span>
             </el-menu-item>
          </template>

        </el-menu>
      </el-aside>
      <el-container>
        <el-header class="header">
          <div class="header-content">
            管理后台
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
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { 
    User, DataLine, Connection, Setting, Document, DocumentCopy, Upload, Money, 
    FullScreen, Monitor, ArrowRight 
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()

// Map string icons to components
const iconMap = {
    'user': User,
    'data-line': DataLine,
    'connection': Connection,
    'setting': Setting,
    'document': Document,
    'document-copy': DocumentCopy,
    'upload': Upload,
    'money': Money,
    'full-screen': FullScreen,
    'dashboard': FullScreen, // Alias
    'monitor': Monitor
}

const getIcon = (name) => iconMap[name] || DataLine

const role = localStorage.getItem('role') || 'user'

const hasPermission = (route) => {
    if (route.meta && route.meta.roles) {
        return route.meta.roles.includes(role)
    }
    return true
}

const filterRoutes = (routes) => {
    const res = []
    routes.forEach(route => {
        const tmp = { ...route }
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
    // Find the main route with children
    const mainRoute = router.options.routes.find(r => r.path === '/')
    if (mainRoute && mainRoute.children) {
        // First filter by hidden
        let visible = mainRoute.children.filter(child => !child.meta?.hidden)
        // Then filter by role
        return filterRoutes(visible)
    }
    return []
})

const activeMenu = computed(() => {
    return route.path
})

const resolvePath = (routeItem, childItem) => {
    let parentPath = routeItem.path.replace(/^\/+|\/+$/g, '')
    
    if (childItem) {
        let childPath = childItem.path.replace(/^\/+|\/+$/g, '')
        return `/${parentPath}/${childPath}`
    }
    
    // If it's a root item (no children displayed or leaf), check if path is empty or /
    if (!parentPath) return '/'
    return '/' + parentPath
}
</script>

<style scoped>
.common-layout {
  height: 100vh;
}
.el-container {
  height: 100%;
}
.aside {
  background-color: #545c64;
  color: white;
}
.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  font-weight: bold;
  font-size: 20px;
  background-color: #434a50;
}
.header {
  background-color: #fff;
  border-bottom: 1px solid #dcdfe6;
  display: flex;
  align-items: center;
}
</style>
