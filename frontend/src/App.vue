<script setup>
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <div id="app">
    <nav class="navbar">
      <div class="nav-brand">
        <img alt="Wails logo" src="./assets/images/logo-universal.png" class="nav-logo" />
        <span>MQTT Bench</span>
      </div>
      <ul class="nav-menu">
        <li class="nav-item">
          <RouterLink to="/" class="nav-link" active-class="active">首页</RouterLink>
        </li>
        <li class="nav-item">
          <RouterLink to="/slaves" class="nav-link" active-class="active">Slave管理</RouterLink>
        </li>
        <li class="nav-item">
          <RouterLink to="/linktest" class="nav-link" active-class="active">链接测试</RouterLink>
        </li>
        <li class="nav-item">
          <RouterLink to="/message" class="nav-link" active-class="active">消息测试</RouterLink>
        </li>
        <li class="nav-item">
          <RouterLink to="/report" class="nav-link" active-class="active">生成报告</RouterLink>
        </li>
      </ul>
    </nav>
    
    <div class="container">
      <RouterView />
    </div>
  </div>
</template>

<style>
.navbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 2rem;
  background-color: #2c3e50;
  color: white;
  box-shadow: 0 2px 4px rgba(0,0,0,.1);
}

.nav-brand {
  display: flex;
  align-items: center;
  font-size: 1.5rem;
  font-weight: bold;
}

.nav-logo {
  height: 40px;
  margin-right: 10px;
}

.nav-menu {
  display: flex;
  list-style: none;
}

.nav-item {
  margin-left: 2rem;
}

.nav-link {
  color: white;
  text-decoration: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.nav-link:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

.nav-link.active {
  background-color: #42b983;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
}
</style>