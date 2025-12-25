import { createRouter, createWebHistory } from 'vue-router'
import HomeView from './views/HomeView.vue'
import SlaveManager from './views/SlaveManager.vue'
import LinkTest from './views/LinkTest.vue'
import Message from './views/Message.vue'
import Report from './views/Report.vue'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: HomeView
  },
  {
    path: '/slaves',
    name: 'Slaves',
    component: SlaveManager
  },
  {
    path: '/linktest',
    name: 'LinkTest',
    component: LinkTest
  },
  {
    path: '/message',
    name: 'Message',
    component: Message
  },
  {
    path: '/report',
    name: 'Report',
    component: Report
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router