<template>
  <div class="message">
    <h1>消息测试</h1>
    <div class="section">
      <h2>连接配置</h2>
      <div class="form-group">
        <label for="brokerUrl">Broker URL:</label>
        <input type="text" id="brokerUrl" v-model="connectionConfig.brokerUrl" placeholder="mqtt://localhost:1883">
      </div>
      
      <div class="form-group">
        <label for="clientId">客户端 ID:</label>
        <input type="text" id="clientId" v-model="connectionConfig.clientId" placeholder="mqtt-client-1">
      </div>
      
      <div class="form-group">
        <label for="username">用户名:</label>
        <input type="text" id="username" v-model="connectionConfig.username">
      </div>
      
      <div class="form-group">
        <label for="password">密码:</label>
        <input type="password" id="password" v-model="connectionConfig.password">
      </div>
      
      <button @click="connect" :disabled="isConnected" class="btn btn-primary">
        {{ isConnected ? '已连接' : '连接' }}
      </button>
      <button @click="disconnect" :disabled="!isConnected" class="btn btn-danger">断开连接</button>
    </div>
    
    <div class="section" v-if="isConnected">
      <h2>消息发布</h2>
      <div class="form-group">
        <label for="topic">主题:</label>
        <input type="text" id="topic" v-model="publishConfig.topic" placeholder="test/topic">
      </div>
      
      <div class="form-group">
        <label for="message">消息内容:</label>
        <textarea id="message" v-model="publishConfig.message" rows="4" placeholder="输入要发布的消息内容"></textarea>
      </div>
      
      <div class="form-group">
        <label for="publishQos">QoS 级别:</label>
        <select id="publishQos" v-model="publishConfig.qos">
          <option value="0">0 - 最多一次</option>
          <option value="1">1 - 至少一次</option>
          <option value="2">2 - 恰好一次</option>
        </select>
      </div>
      
      <button @click="publishMessage" class="btn btn-primary">发布消息</button>
    </div>
    
    <div class="section" v-if="isConnected">
      <h2>消息订阅</h2>
      <div class="form-group">
        <label for="subscribeTopic">订阅主题:</label>
        <input type="text" id="subscribeTopic" v-model="subscribeConfig.topic" placeholder="test/topic">
      </div>
      
      <div class="form-group">
        <label for="subscribeQos">QoS 级别:</label>
        <select id="subscribeQos" v-model="subscribeConfig.qos">
          <option value="0">0 - 最多一次</option>
          <option value="1">1 - 至少一次</option>
          <option value="2">2 - 恰好一次</option>
        </select>
      </div>
      
      <button @click="subscribe" class="btn btn-primary">订阅</button>
      <button @click="unsubscribe" class="btn btn-warning">取消订阅</button>
    </div>
    
    <div class="messages-section" v-if="isConnected">
      <h2>消息历史</h2>
      <div class="message-list">
        <div v-for="(msg, index) in messages" :key="index" class="message-item">
          <div class="message-header">
            <span class="message-topic">主题: {{ msg.topic }}</span>
            <span class="message-time">{{ msg.timestamp }}</span>
          </div>
          <div class="message-content">
            {{ msg.content }}
          </div>
        </div>
        <div v-if="messages.length === 0" class="no-messages">
          <p>暂无消息</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { reactive, ref } from 'vue'

export default {
  name: 'Message',
  setup() {
    const isConnected = ref(false)
    const messages = ref([])
    
    const connectionConfig = reactive({
      brokerUrl: 'mqtt://localhost:1883',
      clientId: 'mqtt-client-1',
      username: '',
      password: ''
    })
    
    const publishConfig = reactive({
      topic: 'test/topic',
      message: 'Hello MQTT!',
      qos: 0
    })
    
    const subscribeConfig = reactive({
      topic: 'test/topic',
      qos: 0
    })
    
    const connect = () => {
      isConnected.value = true
      console.log('连接到 MQTT Broker:', connectionConfig)
    }
    
    const disconnect = () => {
      isConnected.value = false
      messages.value = []
      console.log('断开 MQTT 连接')
    }
    
    const publishMessage = () => {
      if (!isConnected.value) return
      
      console.log('发布消息:', publishConfig)
      // 模拟消息发布
      const newMessage = {
        topic: publishConfig.topic,
        content: publishConfig.message,
        timestamp: new Date().toLocaleTimeString()
      }
      
      // 将发布的消息添加到消息列表
      messages.value.unshift(newMessage)
    }
    
    const subscribe = () => {
      if (!isConnected.value) return
      
      console.log('订阅主题:', subscribeConfig)
      // 模拟订阅成功后的消息接收
      setTimeout(() => {
        const mockMessage = {
          topic: subscribeConfig.topic,
          content: `收到订阅消息: ${new Date().toISOString()}`,
          timestamp: new Date().toLocaleTimeString()
        }
        messages.value.unshift(mockMessage)
      }, 1000)
    }
    
    const unsubscribe = () => {
      console.log('取消订阅主题:', subscribeConfig.topic)
    }
    
    return {
      isConnected,
      messages,
      connectionConfig,
      publishConfig,
      subscribeConfig,
      connect,
      disconnect,
      publishMessage,
      subscribe,
      unsubscribe
    }
  }
}
</script>

<style scoped>
.message {
  padding: 20px;
}

.section {
  background-color: #f8f9fa;
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 30px;
}

.section h2 {
  margin-top: 0;
  color: #2c3e50;
}

.form-group {
  margin-bottom: 15px;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
  font-weight: bold;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  box-sizing: border-box;
}

.form-group textarea {
  resize: vertical;
}

.btn {
  padding: 10px 20px;
  margin-right: 10px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.btn-primary {
  background-color: #42b983;
  color: white;
}

.btn-danger {
  background-color: #dc3545;
  color: white;
}

.btn-warning {
  background-color: #ffc107;
  color: #212529;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.messages-section {
  margin-top: 30px;
}

.message-list {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.message-item {
  padding: 15px;
  border-bottom: 1px solid #eee;
}

.message-item:last-child {
  border-bottom: none;
}

.message-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 5px;
}

.message-topic {
  font-weight: bold;
  color: #42b983;
}

.message-time {
  color: #666;
  font-size: 0.9em;
}

.message-content {
  background-color: white;
  padding: 10px;
  border-radius: 4px;
  border-left: 3px solid #42b983;
}

.no-messages {
  text-align: center;
  padding: 40px;
  color: #666;
}
</style>