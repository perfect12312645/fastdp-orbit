import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import { addCollection, _api } from '@iconify/vue'
import { icons as mdiIcons } from '@iconify-json/mdi'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'
import './styles/index.css'

// 加载 mdi 图标集（离线模式）
addCollection(mdiIcons)

// 禁用 iconify CDN API，完全离线
_api.setFetch(() => Promise.reject(new Error('CDN disabled')))

const app = createApp(App)
const pinia = createPinia()

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(pinia)
app.use(router)
app.use(ElementPlus, { size: 'default' })

app.mount('#app')
