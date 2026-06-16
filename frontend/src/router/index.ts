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
        path: 'gpu',
        name: 'GpuResource',
        component: () => import('@/pages/GpuResource.vue'),
        meta: { title: 'GPU资源管理', icon: 'Cpu' },
      },
      {
        path: 'model-service',
        name: 'ModelService',
        component: () => import('@/pages/ModelService.vue'),
        meta: { title: '大模型服务', icon: 'Connection' },
      },
      {
        path: 'storage',
        name: 'StorageManagement',
        component: () => import('@/pages/StorageManagement.vue'),
        meta: { title: '存储管理', icon: 'Box' },
      },
      {
        path: 'cluster',
        name: 'ClusterDeployment',
        component: () => import('@/pages/ClusterDeployment.vue'),
        meta: { title: '集群部署', icon: 'Share' },
      },
      {
        path: 'workflow',
        name: 'WorkflowManagement',
        component: () => import('@/pages/WorkflowManagement.vue'),
        meta: { title: '工作流', icon: 'Cpu' },
      },
      {
        path: 'workflow/:id/executions/:eid',
        name: 'ExecutionDetail',
        component: () => import('@/pages/ExecutionDetail.vue'),
        meta: { title: '执行详情', icon: 'Cpu', hidden: true },
      },
      {
        path: 'testing',
        name: 'AutoTesting',
        component: () => import('@/pages/AutoTesting.vue'),
        meta: { title: '自动化测试', icon: 'Finished' },
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
