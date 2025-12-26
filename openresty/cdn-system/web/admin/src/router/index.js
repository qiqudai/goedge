import { createRouter, createWebHistory } from 'vue-router'
import Layout from '../views/layout/Layout.vue'

// 0. Public Routes
const publicRoutes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue')
  }
]

// 1. Unified Routes (Admin + User)
export const asyncRoutes = [
  {
    path: '/',
    component: Layout,
    redirect: () => '/dashboard',
    children: [
      // 0. Dashboard
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/dashboard/Index.vue'),
        meta: { title: '仪表盘', icon: 'full-screen', affix: true, roles: ['admin', 'user'] }
      },

      // 1. 节点管理 (Admin Only)
      {
        path: 'node',
        name: 'NodeManagement',
        meta: { title: '节点管理', icon: 'data-line', roles: ['admin'] },
        children: [
          { path: 'list', name: 'NodeList', component: () => import('../views/nodes/List.vue'), meta: { title: '节点列表' } },
          { path: 'groups', name: 'NodeGroups', component: () => import('../views/nodes/groups/List.vue'), meta: { title: '节点分组' } },
          { path: 'dns', name: 'DNS', component: () => import('../views/dns/Index.vue'), meta: { title: 'DNS设置' } },
          { path: 'monitor', name: 'Monitor', component: () => import('../views/settings/Monitor.vue'), meta: { title: '监控配置' } },
          { path: 'realtime', name: 'NodeRealtimeMonitor', component: () => import('../views/nodes/RealtimeMonitor.vue'), meta: { title: '实时监控' } },
          { path: 'groups/:id/resolution', name: 'NodeGroupResolution', component: () => import('../views/nodes/groups/Resolution.vue'), meta: { title: '分组解析配置', hidden: true } }
        ]
      },

      // 2. 全局配置 (Admin Only)
      {
        path: 'global',
        name: 'GlobalConfig',
        meta: { title: '全局配置', icon: 'setting', roles: ['admin'] },
        children: [
          { path: 'firewall', name: 'Firewall', component: () => import('../views/global/Firewall.vue'), meta: { title: '防火墙配置' } },
          { path: 'nginx', name: 'Nginx', component: () => import('../views/global/Nginx.vue'), meta: { title: 'nginx配置' } },
          { path: 'resources', name: 'Resources', component: () => import('../views/global/Resources.vue'), meta: { title: '资源配置' } },
          { path: 'default', name: 'DefaultConfig', component: () => import('../views/global/DefaultConfig.vue'), meta: { title: '默认配置' } },
          { path: 'errors', name: 'ErrorPages', component: () => import('../views/global/ErrorPages.vue'), meta: { title: '错误页面' } }
        ]
      },

      // 3. 网站管理 (Shared)
      {
        path: 'website',
        name: 'WebsiteManagement',
        meta: { title: '网站管理', icon: 'monitor', roles: ['admin', 'user'] },
        children: [
          { path: 'list', name: 'SiteList', component: () => import('../views/website/List.vue'), meta: { title: '网站列表' } },
          { path: 'groups', name: 'SiteGroups', component: () => import('../views/website/Groups.vue'), meta: { title: '网站分组', roles: ['admin', 'user'] } },
          { path: 'resolve', name: 'SiteResolve', component: () => import('../views/website/Resolve.vue'), meta: { title: '解析检测', roles: ['admin'], hidden: true } },
          { path: 'certs', name: 'CertManagement', component: () => import('../views/website/Certs.vue'), meta: { title: '证书管理' } },
          { path: 'purge', name: 'CachePurge', component: () => import('../views/website/Purge.vue'), meta: { title: '刷新预热' } },
          { path: 'rules', name: 'RuleManagement', component: () => import('../views/website/Rules.vue'), meta: { title: '规则管理' } },
          { path: 'monitor', name: 'SiteMonitor', component: () => import('../views/website/Statistics.vue'), meta: { title: '数据统计' } },
          { path: 'logs/block', name: 'BlockLogs', component: () => import('../views/website/BlockLogs.vue'), meta: { title: '拉黑日志' } },
          { path: 'logs/access', name: 'AccessLogs', component: () => import('../views/website/AccessLogs.vue'), meta: { title: '访问日志' } }
        ]
      },

      // 4. 四层转发 (Shared)
      {
        path: 'forward',
        name: 'Forwarding',
        meta: { title: '四层转发', icon: 'connection', roles: ['admin', 'user'] },
        children: [
          { path: 'list', name: 'ForwardList', component: () => import('../views/forward/List.vue'), meta: { title: '转发列表' } },
          { path: 'groups', name: 'ForwardGroups', component: () => import('../views/forward/Groups.vue'), meta: { title: '分组设置' } },
          { path: 'default', name: 'ForwardDefault', component: () => import('../views/forward/Default.vue'), meta: { title: '默认设置' } },
          { path: 'monitor', name: 'ForwardMonitor', component: () => import('../views/forward/Monitor.vue'), meta: { title: '实时监控' } }
        ]
      },

      // 5. 套餐管理 (Shared)
      {
        path: 'plans',
        name: 'PlanManagement',
        meta: { title: '套餐管理', icon: 'money', roles: ['admin', 'user'] },
        children: [
          { path: 'my', name: 'MyPackages', component: () => import('../views/packages/My.vue'), meta: { title: '我的套餐', roles: ['user'] } },
          { path: 'usage', name: 'PackageUsage', component: () => import('../views/packages/Usage.vue'), meta: { title: '用量查询', roles: ['user'] } },
          { path: 'basic', name: 'BasicPlans', component: () => import('../views/plans/Basic.vue'), meta: { title: '基础套餐', roles: ['admin'] } },
          { path: 'sold', name: 'SoldPlans', component: () => import('../views/plans/Sold.vue'), meta: { title: '已售套餐', roles: ['admin'] } }
        ]
      },

      // 6. 系统管理 (Admin Only)
      {
        path: 'system',
        name: 'SystemManagement',
        meta: { title: '系统管理', icon: 'setting', roles: ['admin'] },
        children: [
          { path: 'config', name: 'SystemConfig', component: () => import('../views/settings/System.vue'), meta: { title: '系统配置' } },
          { path: 'tasks', name: 'BackgroundTasks', component: () => import('../views/system/Tasks.vue'), meta: { title: '后台任务' } },
          { path: 'users', name: 'SystemUsers', component: () => import('../views/users/List.vue'), meta: { title: '用户列表' } },
          { path: 'logs', name: 'SystemLogs', component: () => import('../views/logs/Operation.vue'), meta: { title: '系统日志' } },
          { path: 'upgrade', name: 'Upgrade', component: () => import('../views/system/Upgrade.vue'), meta: { title: '维护升级' } },
          { path: 'announcements', name: 'Announcements', component: () => import('../views/system/Announcements.vue'), meta: { title: '公告管理' } },
          { path: 'messages', name: 'Messages', component: () => import('../views/system/Messages.vue'), meta: { title: '消息查询' } }
        ]
      },

      // 7. 账户中心 (User Only)
      {
        path: 'account',
        name: 'AccountCenter',
        meta: { title: '账户中心', icon: 'user', roles: ['user'] },
        children: [
          { path: 'profile', name: 'UserProfile', component: () => import('../views/account/Profile.vue'), meta: { title: '个人资料' } },
          { path: 'recharge', name: 'UserRecharge', component: () => import('../views/account/Recharge.vue'), meta: { title: '账户充值' } },
          { path: 'bills', name: 'UserBills', component: () => import('../views/account/Bills.vue'), meta: { title: '消费记录' } },
          { path: 'logs', name: 'UserLogs', component: () => import('../views/account/Logs.vue'), meta: { title: '日志查询' } },
          { path: 'messages', name: 'UserMessages', component: () => import('../views/account/Messages.vue'), meta: { title: '消息记录' } },
          { path: 'subscribe', name: 'UserSubscribe', component: () => import('../views/account/Subscribe.vue'), meta: { title: '消息订阅' } },
          { path: 'apikey', name: 'UserApiKey', component: () => import('../views/account/ApiKey.vue'), meta: { title: 'API密钥' } }
        ]
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes: publicRoutes.concat(asyncRoutes)
})

// Navigation Guard
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('admin_token')
  const role = localStorage.getItem('role') || 'user'

  if (to.path !== '/login' && !token) {
    next('/login')
  } else {
    if (to.meta && to.meta.roles) {
      if (to.meta.roles.includes(role)) {
        next()
      } else {
        if (role === 'admin') next('/node/list')
        else next('/website/list')
      }
    } else {
      next()
    }
  }
})

export default router

