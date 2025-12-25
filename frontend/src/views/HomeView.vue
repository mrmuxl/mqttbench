<template>
  <div class="home">
    <img alt="Wails logo" src="../assets/images/logo-universal.png" class="logo" />
    <h1>MQTT 基准测试工具</h1>
    <p>欢迎使用 MQTT 基准测试工具</p>
  </div>
</template>

<script>
export default {
  name: 'HomeView'
}
</script>

<style scoped>
.logo {
  display: block;
  width: 30%;
  height: 30%;
  margin: 20px auto;
}

.home {
  text-align: center;
  padding: 20px;
}

h1 {
  color: #42b983;
  margin-bottom: 10px;
}

p {
  color: #666;
  font-size: 18px;
}
</style>