import { createRouter, createWebHistory } from 'vue-router'
import Benchmarks from '../views/Benchmarks.vue'
import Login from '../views/Login.vue'
import BenchmarkCreate from '../views/BenchmarkCreate.vue'
import BenchmarkDetail from '../views/BenchmarkDetail.vue'
import APITokens from '../views/APITokens.vue'
import Users from '../views/Users.vue'
import DebugCalc from '../views/DebugCalc.vue'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/',
    name: 'benchmarks',
    component: Benchmarks
  },
  {
    path: '/benchmarks',
    redirect: '/'
  },
  {
    path: '/benchmarks/new',
    name: 'benchmark-create',
    component: BenchmarkCreate,
    meta: { requiresAuth: true }
  },
  {
    path: '/benchmarks/my',
    name: 'my-benchmarks',
    component: Benchmarks,
    meta: { requiresAuth: true }
  },
  {
    path: '/benchmarks/:id',
    name: 'benchmark-detail',
    component: BenchmarkDetail
  },
  {
    path: '/benchmark/:id',
    redirect: to => {
      return { path: `/benchmarks/${to.params.id}` }
    }
  },
  {
    path: '/login',
    name: 'login',
    component: Login
  },
  {
    path: '/api-tokens',
    name: 'api-tokens',
    component: APITokens,
    meta: { requiresAuth: true }
  },
  {
    path: '/admin/users',
    name: 'admin-users',
    component: Users,
    meta: { requiresAuth: true, requiresAdmin: true }
  },
  {
    path: '/debugcalc',
    name: 'debug-calc',
    component: DebugCalc
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard to prevent logged-in users from accessing login page
// and to protect routes that require authentication or admin privileges
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // Wait for the initial auth check to complete before making routing decisions
  if (!authStore.initialized) {
    await authStore.init()
  }

  // If user is authenticated and trying to access login page, redirect to home
  if (to.name === 'login' && authStore.isAuthenticated) {
    next('/benchmarks')
    return
  }

  // Redirect to login if route requires authentication
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login')
    return
  }

  // Redirect to home if route requires admin but user is not admin
  if (to.meta.requiresAdmin && !authStore.isAdmin) {
    next('/')
    return
  }

  next()
})

export default router
