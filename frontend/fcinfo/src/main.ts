import { createApp } from 'vue'
import './style.css'
import './assets/styles/theme.scss' // 运行时主题变量（亮/暗）
import App from './App.vue'
import router from './router'
import { setUnauthorizedHandler } from './api/request'

// token 失效时统一回登录页，并把当前地址带上以便登录后回跳。
// 用回调注册而不是在 request.ts 里 import router，避免两边循环依赖。
setUnauthorizedHandler(() => {
  if (router.currentRoute.value.name === 'login') return
  router.push({
    name: 'login',
    query: { redirect: router.currentRoute.value.fullPath }
  })
})

createApp(App).use(router).mount('#app')
