import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
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
  }
})