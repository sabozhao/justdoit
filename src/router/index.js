import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import Home from '../views/Home.vue'
import Library from '../views/Library.vue'

import WrongQuestions from '../views/WrongQuestions.vue'
import Exam from '../views/Exam.vue'
import Login from '../views/Login.vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { requiresGuest: true }
  },
  {
    path: '/',
    name: 'Home',
    component: Home,
    meta: { requiresAuth: true }
  },
  {
    path: '/library',
    name: 'Library',
    component: Library,
    meta: { requiresAuth: true }
  },
  {
    path: '/practice',
    name: 'Practice',
    component: () => import('../views/Practice.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/wrong-questions',
    name: 'WrongQuestions',
    component: WrongQuestions,
    meta: { requiresAuth: true }
  },
  {
    path: '/exam/:id',
    name: 'Exam',
    component: Exam,
    props: true,
    meta: { requiresAuth: true }
  },
  {
    path: '/exam/wrong-questions/:bankId',
    name: 'WrongQuestionsExam',
    component: Exam,
    props: true,
    meta: { requiresAuth: true }
  },
  {
    path: '/admin',
    name: 'Admin',
    component: () => import('../views/Admin.vue'),
    meta: { requiresAuth: true, requiresAdmin: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  
  // 初始化认证状态 - 确保用户信息存在
  if (authStore.token && !authStore.user) {
    try {
      await authStore.initAuth()
    } catch (error) {
      console.log('Auth initialization failed:', error)
      authStore.logout()
    }
  }

  const requiresAuth = to.matched.some(record => record.meta.requiresAuth)
  const requiresGuest = to.matched.some(record => record.meta.requiresGuest)

  if (requiresAuth && !authStore.isLoggedIn) {
    // 需要认证但未登录，跳转到登录页
    next('/login')
  } else if (requiresGuest && authStore.isLoggedIn) {
    // 已登录用户访问登录页，跳转到首页
    next('/')
  } else if (to.meta.requiresAdmin && (!authStore.user || !authStore.user.is_admin)) {
    // 需要管理员权限但用户不是管理员
    ElMessage.error('您没有管理员权限')
    next('/')
  } else {
    next()
  }
})

export default router