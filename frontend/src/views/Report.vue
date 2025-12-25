<template>
  <div class="report">
    <h1>测试报告</h1>
    <div class="report-controls">
      <button @click="generateReport" class="btn btn-primary">生成报告</button>
      <button @click="exportReport" class="btn btn-secondary">导出报告</button>
    </div>
    
    <div v-if="reportData" class="report-content">
      <h2>报告概览</h2>
      <div class="report-summary">
        <div class="summary-item">
          <h3>Slave配置数量</h3>
          <p>{{ reportData.slaveCount }}</p>
        </div>
        <div class="summary-item">
          <h3>性能测试数量</h3>
          <p>{{ reportData.performanceCount }}</p>
        </div>
        <div class="summary-item">
          <h3>消息测试数量</h3>
          <p>{{ reportData.messageCount }}</p>
        </div>
      </div>
      
      <div class="report-details">
        <h2>详细信息</h2>
        <div class="detail-section">
          <h3>Slave配置详情</h3>
          <table v-if="reportData.slaves.length > 0">
            <thead>
              <tr>
                <th>ID</th>
                <th>地址</th>
                <th>端口</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="slave in reportData.slaves" :key="slave.id">
                <td>{{ slave.id }}</td>
                <td>{{ slave.address }}</td>
                <td>{{ slave.port }}</td>
              </tr>
            </tbody>
          </table>
          <p v-else>暂无Slave配置</p>
        </div>
      </div>
    </div>
    
    <div v-else class="no-report">
      <p>点击"生成报告"按钮生成测试报告</p>
    </div>
  </div>
</template>

<script>
import { ref } from 'vue'
// 导入Wails绑定的方法
import { GetSlaves, GetPerformanceTests, GetMessageTests } from '../../wailsjs/go/main/App'

export default {
  name: 'Report',
  setup() {
    const reportData = ref(null)
    
    const generateReport = async () => {
      try {
        // 获取所有数据
        const slaves = await GetSlaves()
        const performanceTests = await GetPerformanceTests()
        const messageTests = await GetMessageTests()
        
        // 构建报告数据
        reportData.value = {
          slaveCount: slaves.length,
          performanceCount: performanceTests.length,
          messageCount: messageTests.length,
          slaves: slaves
        }
      } catch (error) {
        console.error('生成报告失败:', error)
      }
    }
    
    const exportReport = () => {
      // 导出报告功能
      alert('导出报告功能待实现')
    }
    
    return {
      reportData,
      generateReport,
      exportReport
    }
  }
}
</script>

<style scoped>
.report {
  padding: 20px;
}

.report-controls {
  margin-bottom: 30px;
}

.report-controls .btn {
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

.btn-secondary {
  background-color: #6c757d;
  color: white;
}

.report-summary {
  display: flex;
  gap: 20px;
  margin-bottom: 30px;
}

.summary-item {
  flex: 1;
  padding: 20px;
  border: 1px solid #ddd;
  border-radius: 4px;
  text-align: center;
}

.summary-item h3 {
  margin-top: 0;
  color: #666;
}

.summary-item p {
  font-size: 24px;
  font-weight: bold;
  color: #42b983;
  margin: 10px 0 0;
}

.report-details {
  margin-top: 30px;
}

.detail-section {
  margin-bottom: 30px;
}

.detail-section h3 {
  border-bottom: 1px solid #eee;
  padding-bottom: 10px;
}

.detail-section table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 10px;
}

.detail-section th,
.detail-section td {
  border: 1px solid #ddd;
  padding: 12px;
  text-align: left;
}

.detail-section th {
  background-color: #f2f2f2;
  font-weight: bold;
}

.no-report {
  text-align: center;
  padding: 40px;
  color: #666;
}
</style>