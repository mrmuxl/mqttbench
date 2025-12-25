<template>
  <div class="slave-manager">
    
    <!-- 配置结果弹窗 -->
    <div v-if="showConfigResult" class="modal">
      <div class="modal-content config-result-modal">
        <span class="close" @click="closeConfigResult">&times;</span>
        <h2>配置下发结果</h2>
        <div class="config-result-content">
          <!-- 使用表格展示配置下发结果 -->
          <table class="result-table" v-if="configResults.length > 0">
            <thead>
              <tr>
                <th>Slave ID</th>
                <th>下发状态</th>
                <th>状态信息</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(result, index) in configResults" :key="index">
                <td>{{ result.slaveId }}</td>
                <td :class="getResultStatusClass(result.successCount, result.failureCount)">
                  {{ getResultStatus(result.successCount, result.failureCount) }}
                </td>
                <td>{{ result.message }}</td>
              </tr>
            </tbody>
          </table>
          <div v-else class="no-results">
            <p>正在等待Slave反馈结果...</p>
          </div>
        </div>
        <button @click="closeConfigResult" class="btn btn-primary">确定</button>
      </div>
    </div>
    
    <!-- 删除确认弹窗 -->
    <div v-if="showDeleteConfirm" class="modal">
      <div class="modal-content">
        <span class="close" @click="cancelDelete">&times;</span>
        <h2>确认删除</h2>
        <p>确定要删除 Slave "{{ slaveToDelete?.name }}" 吗？此操作不可恢复。</p>
        <div class="modal-buttons">
          <button @click="cancelDelete" class="btn btn-secondary">取消</button>
          <button @click="confirmDelete" class="btn btn-danger">删除</button>
        </div>
      </div>
    </div>
    
    <div class="slave-list">
      <div v-if="slaves && slaves.length > 0">
        <p>找到 {{ slaves.length }} 个 slave</p>
        <table>
          <thead>
            <tr>
              <th><input type="checkbox" @change="toggleSelectAll" v-model="selectAll"></th>
              <th>ID</th>
              <th>Name</th>
              <th>IP</th>
              <th>端口</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="slave in slaves" :key="slave.id">
              <td><input type="checkbox" :value="slave.id" v-model="selectedSlaves" :disabled="isSlaveOffline(slave)"></td>
              <td>{{ slave.id }}</td>
              <td>{{ slave.name }}</td>
              <td>{{ slave.slave_host }}</td>
              <td>{{ slave.slave_port }}</td>
              <td :class="getStatusClass(slave.status)">
                <span v-if="!slave.status || slave.status === ''" style="color: purple; font-weight: bold;">[状态为空]</span>
                <span v-else>{{ slave.status }}</span>
              </td>
              <td>
                <button @click="editSlave(slave)" class="btn btn-small btn-warning">配置</button>
                <button @click="deleteSlave(slave)" class="btn btn-small btn-danger">删除</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="no-slaves">
        <p>暂无 Slave 配置</p>
        <p v-if="slaves && Array.isArray(slaves)">Slaves 数组为空</p>
        <p v-else>Slaves 未定义或不是数组</p>
        <!-- 添加调试信息显示 -->
        <p v-if="slaves === null">slaves 值为 null</p>
        <p v-else-if="slaves === undefined">slaves 值为 undefined</p>
        <p v-else-if="Array.isArray(slaves)">slaves 是数组，长度为 {{ slaves.length }}</p>
      </div>
    </div>
    
    <!-- 控制按钮区域 -->
    <div class="controls-bottom">
      <button @click="deployConfig" class="btn btn-primary" :disabled="isDeploying || !hasOnlineSlavesSelected()">
        <span v-if="isDeploying" class="spinner"></span>
        {{ isDeploying ? '下发中...' : '下发配置' }}
      </button>
      <button @click="refreshSlaves" class="btn btn-secondary" :disabled="isRefreshing">
        <span v-if="isRefreshing" class="spinner"></span>
        {{ isRefreshing ? '刷新中...' : '刷新' }}
      </button>
    </div>
    
    <!-- 添加/编辑 Slave 对话框 -->
    <div v-if="showModal" class="modal">
      <div class="modal-content">
        <span class="close" @click="closeModal">&times;</span>
        <h2>{{ editingSlave ? '配置 Slave' : '添加 Slave' }}</h2>
        <form @submit.prevent="saveSlave">
          <div class="form-group horizontal">
            <label for="name">Name:</label>
            <input type="text" id="name" v-model="currentSlave.name" required>
          </div>
          <div class="form-group horizontal">
            <label for="mqtt_host">MQTT IP:</label>
            <input type="text" id="mqtt_host" v-model="currentSlave.mqtt_host" required>
          </div>
          <div class="form-group horizontal" id="mqtt-port-group">
            <label for="mqtt_port">MQTT 端口:</label>
            <input type="number" id="mqtt_port" v-model="currentSlave.mqtt_port" required>
          </div>
          <div class="form-group horizontal">
            <label for="qos">QoS:</label>
            <select id="qos" v-model="currentSlave.qos" required>
              <option value="0">0</option>
              <option value="1">1</option>
              <option value="2">2</option>
            </select>
          </div>
          <div class="form-group horizontal">
            <label for="topic">Sub Topic:</label>
            <input type="text" id="topic" v-model="currentSlave.topic" required>
          </div>
          <div class="form-group horizontal">
            <label for="ack_topic">ACK Topic:</label>
            <input type="text" id="ack_topic" v-model="currentSlave.ack_topic">
          </div>
          <div class="form-group horizontal">
            <label for="client_id">Client ID:</label>
            <input type="text" id="client_id" v-model="currentSlave.client_id">
          </div>
          <div class="form-row">
            <div class="form-group horizontal inline">
              <label for="start">Start:</label>
              <input type="number" id="start" v-model="currentSlave.start" class="short-input">
            </div>
            <div class="form-group horizontal inline">
              <label for="step">Step:</label>
              <input type="number" id="step" v-model="currentSlave.step" class="short-input">
            </div>
          </div>
           <br/>
          <button type="submit" class="btn btn-primary">保存</button>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import { reactive, ref, onMounted, watch } from 'vue'
import { 
  GetSlaves, 
  AddSlave, 
  UpdateSlave, 
  DeleteSlave, 
  DeployConfig, 
  GetConfigResult 
} from '../../wailsjs/go/main/App'

export default {
  name: 'SlaveManagerOptimized',
  setup() {
    // 响应式数据
    const slaves = ref([])
    const showModal = ref(false)
    const showConfigResult = ref(false)
    const showDeleteConfirm = ref(false)
    const configResults = ref([])
    const editingSlave = ref(null)
    const slaveToDelete = ref(null)
    const selectedSlaves = ref([])
    const selectAll = ref(false)
    const isRefreshing = ref(false)
    const isDeploying = ref(false)
    
    // 当前Slave表单数据
    const newSlave = reactive({
      ip: '',
      port: 1883,
      client_id: '',  // 移除默认值 '00001'
      topic: '',
      qos: 0,
      start: 0,
      end: 0,
      ack_topic: 'EEW/ACK/Channel1'
    });
    
    // 创建一个指向newSlave的别名，以便与现有代码兼容
    const currentSlave = newSlave;
    
    /**
     * 组件生命周期钩子
     */
    onMounted(() => {
      refreshSlaves()
    })
    
    /**
     * 监听器
     */
    watch(selectedSlaves, (newVal) => {
      selectAll.value = newVal.length === slaves.value.length && slaves.value.length > 0
    })
    
    /**
     * 状态检查函数
     */
    
    // 判断Slave是否处于离线状态
    const isSlaveOffline = (slave) => {
      return slave.status !== 'online'
    }
    
    // 判断是否有在线的Slave被选中
    const hasOnlineSlavesSelected = () => {
      if (!selectedSlaves.value || selectedSlaves.value.length === 0) {
        return false
      }
      
      return slaves.value.some(slave => 
        selectedSlaves.value.includes(slave.id) && slave.status === 'online'
      )
    }
    
    // 获取状态的CSS类
    const getStatusClass = (status) => {
      return status === 'online' ? 'status-online' : 'status-offline'
    }
    
    // 获取结果状态文本
    const getResultStatus = (successCount, failureCount) => {
      if (failureCount > 0 && successCount === 0) {
        return '失败';
      } else if (successCount > 0 && failureCount === 0) {
        return '成功';
      } else if (successCount > 0 && failureCount > 0) {
        return '部分成功';
      } else if (successCount === 0 && failureCount === 0) {
        // 当成功和失败数量都为0时，表示配置已接收
        return '已接收';
      } else {
        return '未知';
      }
    };
    
    // 获取结果状态的CSS类
    const getResultStatusClass = (successCount, failureCount) => {
      if (failureCount > 0 && successCount === 0) {
        return 'status-failed';
      } else if (successCount > 0 && failureCount === 0) {
        return 'status-success';
      } else if (successCount > 0 && failureCount > 0) {
        return 'status-partial';
      } else if (successCount === 0 && failureCount === 0) {
        // 当成功和失败数量都为0时，表示配置已接收
        return 'status-received';
      } else {
        return 'status-unknown';
      }
    };
    
    /**
     * 选择操作函数
     */
    
    // 全选/取消全选功能
    const toggleSelectAll = () => {
      if (selectAll.value) {
        // 全选，但只选择在线的Slave
        selectedSlaves.value = slaves.value
          .filter(slave => slave.status === 'online')
          .map(slave => slave.id)
      } else {
        // 取消全选
        selectedSlaves.value = []
      }
    }
    
    /**
     * UI操作函数
     */
    
    // 添加Slave UI
    const addSlaveUI = () => {
      editingSlave.value = null
      Object.assign(currentSlave, {
        name: '',
        mqtt_host: '',
        mqtt_port: 1883,
        qos: 0,
        topic: '',
        client_id: '',  // 移除默认值 '00001'
        start: 0,
        step: 50000,
        ack_topic: 'EEW/ACK/Channel1'
      })
      showModal.value = true
    }
    
    // 编辑Slave UI
    const editSlaveUI = (slave) => {
      editingSlave.value = slave
      Object.assign(currentSlave, {
        name: slave.name || '',
        mqtt_host: slave.mqtt_host || '',
        mqtt_port: slave.mqtt_port || 1883,
        qos: slave.qos || 0,
        topic: slave.topic || '',
        client_id: slave.client_id || '',  // 移除 formatClientID 格式化
        start: slave.start || 0,
        step: slave.step || 50000,
        ack_topic: slave.ack_topic || 'EEW/ACK/Channel1'
      })
      showModal.value = true
    }
    
    // 关闭模态框
    const closeModal = () => {
      showModal.value = false
      editingSlave.value = null
    }
    
    /**
     * 删除操作函数
     */
    
    // 删除Slave功能
    const deleteSlave = (slave) => {
      slaveToDelete.value = slave
      showDeleteConfirm.value = true
    }
    
    // 确认删除
    const confirmDelete = async () => {
      try {
        if (slaveToDelete.value) {
          await DeleteSlave(slaveToDelete.value.id)
          console.log('Slave删除成功:', slaveToDelete.value.id)
          // 关闭确认弹窗
          showDeleteConfirm.value = false
          slaveToDelete.value = null
          // 刷新列表
          await refreshSlaves()
        }
      } catch (error) {
        console.error('删除Slave失败:', error)
        alert('删除Slave失败: ' + (error.message || '未知错误'))
        // 关闭确认弹窗
        showDeleteConfirm.value = false
        slaveToDelete.value = null
      }
    }
    
    // 取消删除
    const cancelDelete = () => {
      showDeleteConfirm.value = false
      slaveToDelete.value = null
    }
    
    /**
     * 数据获取函数
     */
    
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
          // 显示错误提示
          alert('无法获取Slave列表：返回数据为空')
          return
        }
        
        if (!Array.isArray(slaveList)) {
          console.log('GetSlaves() 返回的不是数组:', slaveList)
          slaves.value = []
          // 显示错误提示
          alert('无法获取Slave列表：数据格式错误')
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
        
        // 格式化Client ID
        const formattedSlaveList = slaveList.map(slave => ({
          ...slave,
          client_id: slave.client_id  // 移除 formatClientID 格式化
        }))
        
        slaves.value = formattedSlaveList
        console.log('更新后的 slaves.value:', slaves.value)
        
        // 刷新后保持选中状态，但只保留在线的Slave
        selectedSlaves.value = selectedSlaves.value.filter(id => 
          formattedSlaveList && formattedSlaveList.some(slave => slave.id === id && slave.status === 'online')
        )
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
    
    /**
     * 配置操作函数
     */
    
    // 关闭配置结果弹窗
    const closeConfigResult = () => {
      showConfigResult.value = false
      configResults.value = []
    }
    
    // 添加下发配置功能
    const deployConfig = async () => {
      // 如果没有选中的在线slave或正在下发配置，则不处理
      if (!hasOnlineSlavesSelected() || isDeploying.value) {
        return
      }
      
      isDeploying.value = true
      configResults.value = []
      showConfigResult.value = true
      
      try {
        console.log('下发配置到选中的Slave ID:', selectedSlaves.value)
        // 过滤出在线的Slave进行配置下发
        const onlineSlaveIds = slaves.value
          .filter(slave => selectedSlaves.value.includes(slave.id) && slave.status === 'online')
          .map(slave => slave.id)
        
        console.log('实际下发配置的在线Slave ID:', onlineSlaveIds)
        
        // 初始化结果数组
        configResults.value = onlineSlaveIds.map(slaveId => ({
          slaveId: slaveId,
          successCount: 0,
          failureCount: 0,
          message: '正在下发配置...'
        }))
        
        // 调用后端下发配置方法
        await DeployConfig(onlineSlaveIds)
        console.log('配置下发成功')
        
        // 显示成功消息
        if (onlineSlaveIds.length > 0) {
          // 等待一段时间让slave处理配置并返回结果
          await new Promise(resolve => setTimeout(resolve, 3000))
          
          // 获取每个slave的配置结果
          for (let i = 0; i < onlineSlaveIds.length; i++) {
            const slaveId = onlineSlaveIds[i]
            try {
              const result = await GetConfigResult(slaveId)
              if (result) {
                configResults.value[i] = {
                  slaveId: result.slave_id || slaveId,
                  successCount: result.success_count || 0,
                  failureCount: result.failure_count || 0,
                  message: result.message || '配置已下发，请查看Slave端的执行结果'
                }
              } else {
                configResults.value[i] = {
                  slaveId: slaveId,
                  successCount: 0,
                  failureCount: 0,
                  message: '配置已下发，但未收到Slave的反馈结果'
                }
              }
            } catch (error) {
              configResults.value[i] = {
                slaveId: slaveId,
                successCount: 0,
                failureCount: 0,
                message: '配置已下发，但获取结果时出错: ' + (error.message || '未知错误')
              }
            }
          }
        }
      } catch (error) {
        console.error('下发配置失败:', error)
        // 显示错误消息
        configResults.value = [{
          slaveId: 0,
          successCount: 0,
          failureCount: 0,
          message: '配置下发失败: ' + (error.message || '未知错误')
        }]
      } finally {
        // 延迟一小段时间再关闭下发状态，让用户能看到动效
        setTimeout(() => {
          isDeploying.value = false
        }, 500)
      }
    }
    
    /**
     * 保存操作函数
     */
    
    // 保存Slave
    const saveSlave = async () => {
      try {
        let name, mqttHost, mqttPort, clientID, topic, qos, start, step, ackTopic
        
        // 对于编辑操作，如果字段为空则保持数据库中的值
        if (editingSlave.value) {
          name = currentSlave.name || editingSlave.value.name || ''
          mqttHost = currentSlave.mqtt_host || editingSlave.value.mqtt_host || ''
          mqttPort = currentSlave.mqtt_port ? parseInt(currentSlave.mqtt_port) : -1 // 使用-1表示保持原值
          clientID = currentSlave.client_id || editingSlave.value.client_id || ''  // 移除默认值 '00001'
          topic = currentSlave.topic || editingSlave.value.topic || ''
          ackTopic = currentSlave.ack_topic || editingSlave.value.ack_topic || 'EEW/ACK/Channel1'
          // QoS 是一个有效值，即使是0也是有效的，所以我们需要特殊处理
          // 如果currentSlave.qos是数字类型，直接使用；如果是空字符串，则使用-1表示保持原值
          if (currentSlave.qos === '' || currentSlave.qos === null || currentSlave.qos === undefined) {
            qos = -1 // 使用-1表示保持原值
          } else {
            qos = parseInt(currentSlave.qos)
          }
          start = currentSlave.start !== '' ? parseInt(currentSlave.start) : -1 // 使用-1表示保持原值
          step = currentSlave.step !== '' ? parseInt(currentSlave.step) : -1 // 使用-1表示保持原值
        } else {
          // 对于新增操作，使用默认值
          name = currentSlave.name || ''
          mqttHost = currentSlave.mqtt_host || ''
          mqttPort = parseInt(currentSlave.mqtt_port) || 1883
          clientID = currentSlave.client_id || ''  // 移除默认值 '00001'
          topic = currentSlave.topic || ''
          ackTopic = currentSlave.ack_topic || 'EEW/ACK/Channel1'
          qos = parseInt(currentSlave.qos) || 0
          start = parseInt(currentSlave.start) || 0
          step = parseInt(currentSlave.step) || 50000
        }
        
        console.log('保存Slave参数:', editingSlave.value?.id, name, mqttHost, mqttPort, clientID, topic, qos, start, step, ackTopic)
        
        if (editingSlave.value) {
          // 编辑现有 Slave
          await UpdateSlave(
            editingSlave.value.id, 
            name,
            mqttHost,
            mqttPort,
            clientID,
            topic,
            qos,
            start,
            step,
            ackTopic
          )
        } else {
          // 添加新 Slave
          await AddSlave(
            name,
            mqttHost,
            mqttPort,
            clientID,
            topic,
            qos,
            start,
            step,
            ackTopic
          )
        }
        closeModal()
        refreshSlaves()
      } catch (error) {
        console.error('保存Slave失败:', error)
      }
    }
    
    /**
     * 暴露给模板的函数和数据
     */
    return {
      // 数据
      slaves,
      showModal,
      showConfigResult,
      showDeleteConfirm,
      configResults,
      editingSlave,
      slaveToDelete,
      selectedSlaves,
      selectAll,
      currentSlave,
      isRefreshing,
      isDeploying,
      
      // UI操作函数
      addSlave: addSlaveUI,
      editSlave: editSlaveUI,
      closeModal,
      
      // 删除操作函数
      deleteSlave,
      confirmDelete,
      cancelDelete,
      
      // 数据获取函数
      refreshSlaves,
      
      // 配置操作函数
      deployConfig,
      closeConfigResult,
      
      // 保存操作函数
      saveSlave,
      
      // 选择操作函数
      toggleSelectAll,
      
      // 状态检查函数
      getStatusClass,
      isSlaveOffline,
      hasOnlineSlavesSelected,
      
      // 结果状态函数
      getResultStatus,
      getResultStatusClass,
    }
  }
}
</script>

<style scoped>
.slave-manager {
  padding: 20px;
  position: relative;
  min-height: 500px;
  width: 100%;
  box-sizing: border-box;
}

.btn {
  padding: 8px 16px;
  margin-right: 10px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  z-index: 10;
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
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

.btn-warning {
  background-color: #ffc107;
  color: #212529;
}

.btn-danger {
  background-color: #dc3545;
  color: white;
}

.btn-small {
  padding: 4px 8px;
  font-size: 12px;
  margin-right: 5px;
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

/* 选择框样式 */
.slave-list th:first-child,
.slave-list td:first-child {
  text-align: center;
  width: 40px;
}

.slave-list input[type="checkbox"] {
  transform: scale(1.2);
  cursor: pointer;
}

.slave-list input[type="checkbox"]:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.no-slaves {
  text-align: center;
  padding: 40px;
  color: #666;
}

.modal {
  display: block;
  position: fixed;
  z-index: 1000;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
}

.modal-content {
  background-color: #fefefe;
  margin: 15% auto;
  padding: 20px;
  border: 1px solid #888;
  width: 500px;
  max-width: 90%;
  position: relative;
  z-index: 1001;
}

/* 配置结果弹窗样式 */
.config-result-modal {
  width: 800px;
  max-width: 95%;
  max-height: 80vh;
  overflow-y: auto;
}



.config-result-content {
  margin: 20px 0;
}

.slave-result-item {
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 15px;
  margin-bottom: 15px;
  background-color: #f9f9f9;
}

.slave-result-item h3 {
  margin-top: 0;
  color: #2c3e50;
  border-bottom: 1px solid #eee;
  padding-bottom: 8px;
}

.slave-result-item p {
  margin: 8px 0;
}

.no-results {
  text-align: center;
  padding: 20px;
  color: #666;
}

.close {
  color: #aaa;
  float: right;
  font-size: 28px;
  font-weight: bold;
  position: absolute;
  right: 10px;
  top: 0;
  z-index: 1002;
}

.close:hover,
.close:focus {
  color: black;
  text-decoration: none;
  cursor: pointer;
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
.form-group select {
  width: 100%;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  box-sizing: border-box;
  text-align: left;
}

.form-group select {
  height: 36px;
}

/* 水平布局的表单组 */
.form-group.horizontal {
  display: flex;
  align-items: center;
}

.form-group.horizontal label {
  width: 100px;
  margin-bottom: 0;
  margin-right: 10px;
  text-align: right;
}

.form-group.horizontal input,
.form-group.horizontal select {
  flex: 1;
  width: auto;
  text-align: left;
}

/* 短输入框 */
.short-input {
  width: 100px !important;
}

/* 表单行容器 */
.form-row {
  display: flex;
  gap: 20px;
  align-items: center;
}

.form-row .form-group.horizontal {
  flex: 1;
  margin-bottom: 0;
}

.form-row .form-group.horizontal label {
  width: 100px;
  text-align: right;
  margin-bottom: 0;
  margin-right: 10px;
}

.form-row .form-group.horizontal input,
.form-row .form-group.horizontal select {
  text-align: left;
}

/* 控制按钮区域样式 - 紧贴表格 */
.controls-bottom {
  margin-top: 20px;
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  z-index: 5;
  position: relative;
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

/* 配置结果状态样式 */
.status-success {
  color: green;
  font-weight: bold;
}

.status-failed {
  color: red;
  font-weight: bold;
}

.status-partial {
  color: orange;
  font-weight: bold;
}

.status-received {
  color: blue;
  font-weight: bold;
}

.status-unknown {
  color: gray;
  font-weight: bold;
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

/* 按钮点击效果 */
.btn:active {
  transform: scale(0.98);
  transition: transform 0.1s ease;
}

/* 按钮悬停效果 */
.btn:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0,0,0,0.2);
  transition: all 0.2s ease;
}

/* 删除确认弹窗按钮 */
.modal-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
}

/* 配置结果表格样式 */
.result-table {
  width: 100%;
  border-collapse: collapse;
  margin: 15px 0;
  table-layout: fixed;
}

.result-table th,
.result-table td {
  border: 1px solid #ddd;
  padding: 12px;
  text-align: left;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.result-table th {
  background-color: #f2f2f2;
  font-weight: bold;
}

.result-table td {
  max-width: 0; /* 配合table-layout: fixed实现均匀分配列宽 */
}

.result-table tr:nth-child(even) {
  background-color: #f9f9f9;
}

.result-table tr:hover {
  background-color: #f5f5f5;
}

/* 为不同列设置不同的宽度比例 */
.result-table th:nth-child(1),
.result-table td:nth-child(1) { width: 20%; } /* Slave ID */
.result-table th:nth-child(2),
.result-table td:nth-child(2) { width: 20%; } /* 下发状态 */
.result-table th:nth-child(3),
.result-table td:nth-child(3) { width: 60%; } /* 状态信息 */
</style>