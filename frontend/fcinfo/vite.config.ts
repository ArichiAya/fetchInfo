import type { IncomingMessage } from 'node:http'
import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

/**
 * /login 和 /register 既是 SPA 路由又是后端接口，而 Vite 的代理是按前缀匹配的。
 * 浏览器直接访问 /login（刷新页面、手输网址）是一次 Accept: text/html 的 GET，
 * 不拦的话会被整个甩给后端，SPA 根本加载不出来。
 * 这里把 HTML 导航放行到 index.html，只让 POST 的 API 调用走代理。
 */
const bypassHtmlNavigation = (req: IncomingMessage) =>
  req.headers.accept?.includes('text/html') ? '/index.html' : undefined

export default defineConfig(({ mode }) => {
  // 后端地址：默认指向 analysis/cmd/server（端口 8555，见 backend/analysis/cmd/server/main.go）
  // 要改成别的后端，在 frontend/fcinfo/.env.local 里写 VITE_API_TARGET=http://host:port
  const env = loadEnv(mode, process.cwd(), '')
  const apiTarget = env.VITE_API_TARGET || 'http://127.0.0.1:8555'

  return {
    plugins: [
      vue(),
      // 自动导入 Vue 核心 API (ref, reactive) 与 Element Plus 反馈组件 (ElMessage 等)
      AutoImport({
        imports: ['vue', 'vue-router', 'pinia'],
        resolvers: [ElementPlusResolver()],
        dts: 'auto-imports.d.ts'
      }),
      // 自动导入 Element Plus 视图组件
      Components({
        resolvers: [ElementPlusResolver()],
        dts: 'components.d.ts'
      })
    ],
    resolve: {
      alias: {
        // 配置 @ 指向 src 目录
        '@': fileURLToPath(new URL('./src', import.meta.url))
      }
    },
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: `@use "@/assets/styles/variables.scss" as *;` // 注意末尾分号不可省略
        }
      }
    },
    server: {
      proxy: {
        // 前端只认 /api 前缀（见 docs/api.md），真实地址在这里转发
        '/api': {
          target: apiTarget,
          changeOrigin: true
        },
        // 登录接口在 /api 之外（backend/analysis/cmd/server 的 router.POST("/login")），
        // 但同时又是前端的 /login 路由，所以要按 GET/POST 分流
        '/login': {
          target: apiTarget,
          changeOrigin: true,
          bypass: bypassHtmlNavigation
        },
        // 注册与 /login 平级，同样是接口 + 避免直接访问时被代理吞掉
        '/register': {
          target: apiTarget,
          changeOrigin: true,
          bypass: bypassHtmlNavigation
        }
      }
    }
  }
})
