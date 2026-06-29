import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { isLoggedIn } from '@/utils/auth'

/** 路由配置 */
const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/pages/Login.vue'),
    meta: { title: '登录', requiresAuth: false },
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/pages/Dashboard.vue'),
        meta: { title: '仪表盘', icon: 'Odometer' },
      },
      {
        path: 'node',
        name: 'NodeManagement',
        component: () => import('@/pages/NodeManagement.vue'),
        meta: { title: '节点管理', icon: 'Monitor' },
      },

      {
        path: 'storage',
        name: 'StorageManagement',
        component: () => import('@/pages/StorageManagement.vue'),
        meta: { title: '存储管理', icon: 'Box' },
      },
      {
        path: 'workflow',
        name: 'WorkflowManagement',
        component: () => import('@/pages/WorkflowManagement.vue'),
        meta: { title: '工作流', icon: 'Cpu' },
      },
      {
        path: 'workflow/:id/canvas',
        name: 'WorkflowCanvas',
        component: () => import('@/pages/WorkflowCanvas.vue'),
        meta: { title: '工作流编排', icon: 'Cpu', hidden: true },
      },
      {
        path: 'workflow/:id/executions/:eid',
        name: 'ExecutionDetail',
        component: () => import('@/pages/ExecutionDetail.vue'),
        meta: { title: '执行详情', icon: 'Cpu', hidden: true },
      },
      {
        path: 'stages',
        name: 'StageManagement',
        component: () => import('@/pages/StageManagement.vue'),
        meta: { title: '阶段管理', icon: 'View' },
      },
      {
        path: 'global-variables',
        name: 'GlobalVariableManagement',
        component: () => import('@/pages/GlobalVariableManagement.vue'),
        meta: { title: '全局变量', icon: 'Coin' },
      },
      {
        path: 'hook-templates',
        name: 'HookTemplateManagement',
        component: () => import('@/pages/HookTemplateManagement.vue'),
        meta: { title: '钩子管理', icon: 'Hook' },
      },
      {
        path: 'workflow-templates',
        name: 'WorkflowTemplateManagement',
        component: () => import('@/pages/WorkflowTemplateManagement.vue'),
        meta: { title: '配置模板', icon: 'Document' },
      },
      {
        path: 'solution-library',
        name: 'SolutionLibraryManagement',
        component: () => import('@/pages/SolutionLibraryManagement.vue'),
        meta: { title: '方案库', icon: 'Package' },
      },
      {
        path: 'settings',
        name: 'SystemSettings',
        component: () => import('@/pages/SystemSettings.vue'),
        meta: { title: '系统设置', icon: 'Setting' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

/** 路由守卫：未登录跳转登录页 */
router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title || '首页'} - fastdp-orbit`

  if (to.meta.requiresAuth !== false && !isLoggedIn()) {
    next('/login')
  } else if (to.path === '/login' && isLoggedIn()) {
    next('/')
  } else {
    next()
  }
})

export default router
