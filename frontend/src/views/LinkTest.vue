<template>
  <div class="performance">
    <h1>链接测试</h1>
    <div class="slave-list">
      <div v-if="slaves && slaves.length > 0">
        <p>找到 {{ slaves.length }} 个 slave</p>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>状态</th>
              <th>连接数</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="slave in slaves" :key="slave.id">
              <td>{{ slave.name }}</td>
              <td :class="getStatusClass(slave.status)">
                <span v-if="!slave.status || slave.status === ''" style="color: purple; font-weight: bold;">[状态为空]</span>
                <span v-else>{{ slave.status }}</span>
              </td>
              <td>{{ slave.connections || 0 }}</td>
              <td>
                <button @click="startSlave(slave)" class="btn btn-small btn-primary" :disabled="isSlaveOffline(slave)">
                  启动
                </button>
                <button @click="stopSlave(slave)" class="btn btn-small btn-danger" :disabled="isSlaveOffline(slave)">
                  停止
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="no-slaves">
        <p>暂无 Slave 信息</p>
        <p v-if="slaves && Array.isArray(slaves)">Slaves 数组为空</p>
        <p v-else>Slaves 未定义或不是数组</p>
      </div>
    </div>
    
    <!-- 控制按钮区域 -->
    <div class="controls-bottom">
      <button @click="refreshSlaves" class="btn btn-secondary" :disabled="isRefreshing">
        <span v-if="isRefreshing" class="spinner"></span>
        {{ isRefreshing ? '刷新中...' : '刷新' }}
      </button>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { GetSlaves, StartSlave, StopSlave } from '../../wailsjs/go/main/App'

export default {
  name: 'LinkTest',
  setup() {
    const slaves = ref([])
    const isRefreshing = ref(false)
    
    // 判断Slave是否处于离线状态
    const isSlaveOffline = (slave) => {
      return slave.status !== 'online'
    }
    
    // 获取状态的CSS类
    const getStatusClass = (status) => {
      return status === 'online' ? 'status-online' : 'status-offline'
    }
    
    // 启动Slave
    const startSlave = async (slave) => {
      try {
        console.log('启动Slave:', slave.name, 'ID:', slave.id)
        // 检查Slave是否在线
        if (isSlaveOffline(slave)) {
          alert(`Slave ${slave.name} 不在线，无法启动`)
          return
        }
        
        // 调用后端StartSlave方法，传入slave ID
        await StartSlave(slave.id)
        console.log('Slave启动命令已发送:', slave.name)
        alert(`Slave ${slave.name} 启动命令已发送`)
        
        // 刷新列表以更新状态
        await refreshSlaves()
      } catch (error) {
        console.error('启动Slave失败:', error)
        console.error('错误类型:', typeof error)
        console.error('错误信息:', error.message || error)
        alert('启动Slave失败: ' + (error.message || '未知错误'))
      }
    }
    
    // 停止Slave
    const stopSlave = async (slave) => {
      try {
        console.log('停止Slave:', slave.name, 'ID:', slave.id)
        // 检查Slave是否在线
        if (isSlaveOffline(slave)) {
          alert(`Slave ${slave.name} 不在线，无法停止`)
          return
        }
        
        // 调用后端StopSlave方法，传入slave ID
        console.log('调用后端StopSlave方法，传入slave ID:', slave.id)
        await StopSlave(slave.id)
        console.log('Slave停止命令已发送:', slave.name)
        alert(`Slave ${slave.name} 停止命令已发送`)
        
        // 刷新列表以更新状态
        await refreshSlaves()
      } catch (error) {
        console.error('停止Slave失败:', error)
        console.error('错误类型:', typeof error)
        console.error('错误信息:', error.message || error)
        alert('停止Slave失败: ' + (error.message || '未知错误'))
      }
    }
    
    // 刷新Slaves列表
    const refreshSlaves = async () => {
      // 如果正在刷新，则不处理重复点击
      if (isRefreshing.value) {
        console.log('正在刷新中，跳过重复请求')
        return
      }
      
      console.log('开始刷新 slaves...')
      isRefreshing.value = true
      
      try {
        console.log('调用 GetSlaves() 方法...')
        
        const slaveList = await GetSlaves()
        console.log('GetSlaves() 返回结果:', slaveList)
        console.log('slaveList 类型:', typeof slaveList)
        console.log('slaveList 是否为数组:', Array.isArray(slaveList))
        
        // 检查返回值
        if (slaveList === null || slaveList === undefined) {
          console.log('GetSlaves() 返回 null 或 undefined')
          slaves.value = []
          return
        }
        
        if (!Array.isArray(slaveList)) {
          console.log('GetSlaves() 返回的不是数组:', slaveList)
          slaves.value = []
          return
        }
        
        console.log('获取到的Slave列表长度:', slaveList.length)
        
        // 添加调试信息
        if (slaveList.length > 0) {
          console.log('Slave详细信息:')
          slaveList.forEach((slave, index) => {
            console.log(`Slave ${index}:`, slave)
            console.log(`ID: ${slave.id}, Name: ${slave.name}, Status: ${slave.status}, SlaveHost: ${slave.slave_host}, SlavePort: ${slave.slave_port}, UpdatedAt: ${slave.updated_at}`)
          })
        } else {
          console.log('Slave列表为空')
        }
        
        slaves.value = slaveList
        console.log('更新后的 slaves.value:', slaves.value)
      } catch (error) {
        console.error('获取Slave列表失败:', error)
        console.error('错误类型:', typeof error)
        console.error('错误信息:', error.message || error)
        // 即使出错也确保slaves.value是一个数组
        slaves.value = []
        // 显示错误提示
        alert('获取Slave列表失败: ' + (error.message || '未知错误') + '，请检查后端服务是否正常运行。')
      } finally {
        // 延迟一小段时间再关闭刷新状态，让用户能看到动效
        setTimeout(() => {
          isRefreshing.value = false
          console.log('刷新完成')
        }, 500)
      }
    }
    
    // 组件挂载时刷新数据
    onMounted(() => {
      refreshSlaves()
    })
    
    return {
      slaves,
      isRefreshing,
      refreshSlaves,
      getStatusClass,
      startSlave,
      stopSlave,
      isSlaveOffline
    }
  }
}
</script>

<style scoped>
.performance {
  padding: 20px;
}

.performance h1 {
  color: #42b983;
  margin-bottom: 20px;
  text-align: center;
}

.slave-list {
  width: 100%;
  overflow-x: auto;
}

.slave-list table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 20px;
  font-size: 16px;
  font-weight: 500;
  table-layout: fixed;
}

.slave-list th,
.slave-list td {
  border: 1px solid #333;
  padding: 12px;
  text-align: left;
  word-wrap: break-word;
}

.slave-list th {
  background-color: #2c3e50;
  color: white;
  font-weight: bold;
  text-transform: uppercase;
}

.slave-list td {
  background-color: #ecf0f1;
  color: #2c3e50;
  font-weight: 500;
}

.slave-list tr:nth-child(even) td {
  background-color: #bdc3c7;
}

.no-slaves {
  text-align: center;
  padding: 40px;
  color: #666;
}

/* 状态样式 */
.status-online {
  color: green;
  font-weight: bold;
}

.status-offline {
  color: red;
  font-weight: bold;
}

/* 按钮样式 */
.btn {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-right: 5px;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background-color: #42b983;
  color: white;
}

.btn-secondary {
  background-color: #6c757d;
  color: white;
}

.btn-danger {
  background-color: #dc3545;
  color: white;
}

.btn-small {
  padding: 4px 8px;
  font-size: 12px;
}

.btn:hover:not(:disabled) {
  opacity: 0.9;
}

/* 控制按钮区域样式 */
.controls-bottom {
  margin-top: 20px;
  display: flex;
  gap: 10px;
  justify-content: flex-end;
}

/* 刷新动效 */
.spinner {
  width: 12px;
  height: 12px;
  border: 2px solid transparent;
  border-top: 2px solid white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-right: 8px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>