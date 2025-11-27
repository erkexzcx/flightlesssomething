import { createRouter, createWebHistory } from 'vue-router'
import Benchmarks from '../views/Benchmarks.vue'
import Login from '../views/Login.vue'
import BenchmarkCreate from '../views/BenchmarkCreate.vue'
import BenchmarkDetail from '../views/BenchmarkDetail.vue'
import MyBenchmarks from '../views/MyBenchmarks.vue'
import APITokens from '../views/APITokens.vue'
import Users from '../views/Users.vue'
import Logs from '../views/Logs.vue'
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
    component: BenchmarkCreate
  },
  {
    path: '/benchmarks/my',
    name: 'my-benchmarks',
    component: MyBenchmarks
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
    component: APITokens
  },
  {
    path: '/admin/users',
    name: 'admin-users',
    component: Users
  },
  {
    path: '/admin/logs',
    name: 'admin-logs',
    component: Logs
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard to prevent logged-in users from accessing login page
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  
  // If user is authenticated and trying to access login page, redirect to home
  if (to.path === '/login' && authStore.isAuthenticated) {
    next('/benchmarks')
  } else {
    next()
  }
})

export default router
