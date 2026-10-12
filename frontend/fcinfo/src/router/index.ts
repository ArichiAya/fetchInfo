import { createRouter, createWebHistory } from 'vue-router'
import { getToken } from '@/api/request'
import MainPage from '@/components/MainPage.vue'
import LoginView from '@/views/LoginView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
    { path: '/', name: 'home', component: MainPage },
    // 其余路径一律回首页，去留交给守卫判断
    { path: '/:pathMatch(.*)*', redirect: '/' }
  ]
})

/**
 * 登录守卫：
 * - 没有 token 一律回登录页，并把原地址挂在 redirect 上以便登录后回跳
 * - 已登录还去 /login 就送回首页
 */
router.beforeEach((to) => {
  const authed = Boolean(getToken())
  if (!to.meta.public && !authed) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && authed) {
    return { name: 'home' }
  }
  return true
})

export default router
