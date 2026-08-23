// 固件升级管理平台前端 JavaScript

// API 基础路径
const API_BASE = '/api';

// 工具函数：API 请求
async function apiRequest(method, path, data = null, options = {}) {
    const url = `${API_BASE}${path}`;
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers
    };

    const config = {
        method,
        headers,
    };

    if (data) {
        if (data instanceof FormData) {
            config.body = data;
            delete config.headers['Content-Type'];
        } else {
            config.body = JSON.stringify(data);
        }
    }

    try {
        const response = await fetch(url, config);
        const result = await response.json();

        if (response.status >= 400) {
            throw new Error(result.error || result.message || '请求失败');
        }

        return result;
    } catch (error) {
        console.error('API Error:', error);
        throw error;
    }
}

// 通知提示
function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    container.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(100%)';
        toast.style.transition = 'all 0.3s';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// 路由处理
const routes = {
    'dashboard': { title: '仪表盘', load: loadDashboard },
    'models': { title: '设备型号', load: loadModels },
    'devices': { title: '设备管理', load: loadDevices },
    'firmware': { title: '固件管理', load: loadFirmware },
    'tasks': { title: '升级任务', load: loadTasks },
    'history': { title: '升级历史', load: loadHistory },
    'stats': { title: '统计分析', load: loadStats },
    'health': { title: '系统状态', load: loadHealth }
};

// 切换视图
function switchView(viewName) {
    // 隐藏所有视图
    document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
    
    // 显示目标视图
    const view = document.getElementById(`view-${viewName}`);
    if (view) {
        view.classList.remove('hidden');
    }

    // 更新导航高亮
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
        if (item.dataset.view === viewName) {
            item.classList.add('active');
        }
    });

    // 更新面包屑
    if (routes[viewName]) {
        document.getElementById('breadcrumb').textContent = routes[viewName].title;
        // 加载视图数据
        routes[viewName].load();
    }
}

// 解析 URL hash 并路由
function handleRoute() {
    const hash = window.location.hash.substring(1);
    const viewName = hash || 'dashboard';
    if (routes[viewName]) {
        switchView(viewName);
    } else {
        switchView('dashboard');
    }
}

// 模态框操作
function openModal(id) {
    document.getElementById(id).classList.remove('hidden');
}

function closeModal(id) {
    document.getElementById(id).classList.add('hidden');
    // 重置表单
    const form = document.querySelector(`#${id} form`);
    if (form) {
        form.reset();
    }
}

// 渲染条形图
function renderBarChart(containerId, data, title = '') {
    const container = document.getElementById(containerId);
    if (!container) return;

    if (data.length === 0) {
        container.innerHTML = '<div style="color:#909399">暂无数据</div>';
        return;
    }

    const maxValue = Math.max(...data.map(d => d.count));
    
    let html = '<div class="bar-chart">';
    data.forEach(item => {
        const percentage = maxValue > 0 ? (item.count / maxValue * 100) : 0;
        html += `
            <div class="bar-item">
                <div class="bar-label" title="${item.name}">${truncateText(item.name, 10)}</div>
                <div class="bar-track">
                    <div class="bar-fill" style="width: ${percentage}%"></div>
                </div>
                <div class="bar-value">${item.count}</div>
            </div>
        `;
    });
    html += '</div>';
    container.innerHTML = html;
}

// 截断文本
function truncateText(text, maxLen) {
    if (!text) return '';
    return text.length > maxLen ? text.substring(0, maxLen) + '...' : text;
}

// 格式化状态
function getStatusBadge(status) {
    const map = {
        'online': '<span class="status-badge status-online">在线</span>',
        'offline': '<span class="status-badge status-offline">离线</span>',
        'upgrading': '<span class="status-badge status-upgrading">升级中</span>',
        'error': '<span class="status-badge status-error">错误</span>',
        'pending': '<span class="status-badge status-pending">等待</span>',
        'running': '<span class="status-badge status-running">执行中</span>',
        'completed': '<span class="status-badge status-completed">已完成</span>',
        'failed': '<span class="status-badge status-failed">已失败</span>',
        'cancelled': '<span class="status-badge status-cancelled">已取消</span>',
        'success': '<span class="status-badge status-completed">成功</span>',
        'in_progress': '<span class="status-badge status-running">进行中</span>',
        'rollback': '<span class="status-badge status-warning">回滚中</span>'
    };
    return map[status] || `<span class="status-badge">${status}</span>`;
}

// 格式化文件大小
function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    const k = 1024;
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + units[i];
}

// ============ 仪表盘 ============

async function loadDashboard() {
    try {
        const result = await apiRequest('GET', '/stats/dashboard');
        const data = result.data;

        document.getElementById('stat-total-devices').textContent = data.total_devices || 0;
        document.getElementById('stat-online').textContent = data.online_devices || 0;
        document.getElementById('stat-offline').textContent = data.offline_devices || 0;
        document.getElementById('stat-total-models').textContent = data.total_models || 0;
        document.getElementById('stat-total-firmware').textContent = data.total_firmware || 0;
        document.getElementById('stat-active-tasks').textContent = data.active_tasks || 0;
        document.getElementById('stat-success-rate').textContent = (data.success_rate || 0).toFixed(1) + '%';
        document.getElementById('stat-pending').textContent = data.pending_upgrades || 0;

        // 版本分布图
        const versionData = Object.entries(data.version_distribution || {}).map(([name, count]) => ({ name, count }));
        renderBarChart('chart-version-dist', versionData);

        // 型号分布图
        const modelData = Object.entries(data.model_distribution || {}).map(([name, count]) => ({ name, count }));
        renderBarChart('chart-model-dist', modelData);

        // 最近任务
        const tasksResult = await apiRequest('GET', '/tasks?page=1&page_size=5');
        const tasks = tasksResult.data.list || [];
        const tbody = document.querySelector('#table-recent-tasks tbody');
        if (tasks.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" class="loading">暂无任务</td></tr>';
        } else {
            tbody.innerHTML = tasks.map(t => `
                <tr>
                    <td>${t.id}</td>
                    <td>${t.name}</td>
                    <td>${getStatusBadge(t.status)}</td>
                    <td>
                        <div class="progress-bar">
                            <div class="progress-bar-inner" style="width:${t.progress}%"></div>
                        </div>
                        <div class="progress-text">${t.progress}%</div>
                    </td>
                </tr>
            `).join('');
        }
    } catch (error) {
        showToast('加载仪表盘数据失败: ' + error.message, 'error');
    }
}

// ============ 设备型号 ============

async function loadModels() {
    try {
        const result = await apiRequest('GET', '/models?page=1&page_size=100');
        const list = result.data.list || [];
        const tbody = document.querySelector('#table-models tbody');

        if (list.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="loading">暂无型号</td></tr>';
            return;
        }

        tbody.innerHTML = list.map(m => `
            <tr>
                <td>${m.id}</td>
                <td>${m.name}</td>
                <td>${m.manufacturer}</td>
                <td>${m.hardware_version}</td>
                <td>${m.device_count || 0}</td>
                <td>${m.is_active ? '<span class="status-badge status-online">启用</span>' : '<span class="status-badge status-offline">禁用</span>'}</td>
                <td>
                    <div class="action-btns">
                        <button class="btn btn-sm btn-secondary" onclick="editModel(${m.id})">编辑</button>
                        <button class="btn btn-sm btn-danger" onclick="deleteModel(${m.id})">删除</button>
                    </div>
                </td>
            </tr>
        `).join('');
    } catch (error) {
        showToast('加载型号列表失败: ' + error.message, 'error');
    }
}

function submitModel(event) {
    event.preventDefault();
    const id = document.getElementById('model-id').value;
    const data = {
        name: document.getElementById('model-name').value,
        manufacturer: document.getElementById('model-manufacturer').value,
        hardware_version: document.getElementById('model-version').value,
        description: document.getElementById('model-desc').value
    };

    const url = id ? `/models/${id}` : '/models';
    const method = id ? 'PUT' : 'POST';

    apiRequest(method, url, data).then(() => {
        showToast(id ? '更新成功' : '创建成功', 'success');
        closeModal('modal-model');
        loadModels();
    }).catch(error => showToast(error.message, 'error'));
}

function editModel(id) {
    apiRequest('GET', `/models/${id}`).then(result => {
        const m = result.data;
        document.getElementById('model-id').value = m.id;
        document.getElementById('model-name').value = m.name;
        document.getElementById('model-manufacturer').value = m.manufacturer;
        document.getElementById('model-version').value = m.hardware_version;
        document.getElementById('model-desc').value = m.description || '';
        document.getElementById('modal-model-title').textContent = '编辑型号';
        openModal('modal-model');
    }).catch(error => showToast(error.message, 'error'));
}

function deleteModel(id) {
    if (!confirm('确定要删除此型号吗？')) return;
    apiRequest('DELETE', `/models/${id}`).then(() => {
        showToast('删除成功', 'success');
        loadModels();
    }).catch(error => showToast(error.message, 'error'));
}

// ============ 设备管理 ============

async function loadDeviceFormOptions() {
    try {
        const result = await apiRequest('GET', '/models?page=1&page_size=100');
        const models = result.data.list || [];
        const selects = ['device-modelid', 'firmware-modelid', 'task-modelid', 'filter-model'];
        selects.forEach(selectId => {
            const select = document.getElementById(selectId);
            if (select) {
                select.innerHTML = selectId === 'filter-model'
                    ? '<option value="">所有型号</option>' + models.map(m => `<option value="${m.id}">${m.name}</option>`).join('')
                    : models.map(m => `<option value="${m.id}">${m.name}</option>`).join('');
            }
        });
    } catch (error) {
        console.error('Failed to load model options:', error);
    }
}

async function loadDevices() {
    try {
        const modelId = document.getElementById('filter-model')?.value || '';
        const status = document.getElementById('filter-status')?.value || '';
        const params = new URLSearchParams({ page: '1', page_size: '100' });
        if (modelId) params.set('model_id', modelId);
        if (status) params.set('status', status);

        const result = await apiRequest('GET', `/devices?${params}`);
        const list = result.data.list || [];
        const tbody = document.querySelector('#table-devices tbody');

        if (list.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="loading">暂无设备</td></tr>';
            return;
        }

        tbody.innerHTML = list.map(d => `
            <tr>
                <td>${d.device_id}</td>
                <td>${d.name}</td>
                <td>${d.model_name || '-'}</td>
                <td>${d.current_fw_version || '-'}</td>
                <td>${getStatusBadge(d.status)}</td>
                <td>${d.ip_address || '-'}</td>
                <td>
                    <div class="action-btns">
                        <button class="btn btn-sm btn-danger" onclick="deleteDevice(${d.id})">删除</button>
                    </div>
                </td>
            </tr>
        `).join('');
    } catch (error) {
        showToast('加载设备列表失败: ' + error.message, 'error');
    }
}

function searchDevices() {
    const keyword = document.getElementById('search-device').value;
    if (!keyword.trim()) {
        loadDevices();
        return;
    }

    apiRequest('GET', `/devices/search?keyword=${encodeURIComponent(keyword)}&page=1&page_size=100`)
        .then(result => {
            const list = result.data.list || [];
            const tbody = document.querySelector('#table-devices tbody');
            tbody.innerHTML = list.map(d => `
                <tr>
                    <td>${d.device_id}</td>
                    <td>${d.name}</td>
                    <td>${d.model_name || '-'}</td>
                    <td>${d.current_fw_version || '-'}</td>
                    <td>${getStatusBadge(d.status)}</td>
                    <td>${d.ip_address || '-'}</td>
                    <td>
                        <div class="action-btns">
                            <button class="btn btn-sm btn-danger" onclick="deleteDevice(${d.id})">删除</button>
                        </div>
                    </td>
                </tr>
            `).join('');
        })
        .catch(error => showToast(error.message, 'error'));
}

function submitDevice(event) {
    event.preventDefault();
    const data = {
        device_id: document.getElementById('device-deviceid').value,
        model_id: parseInt(document.getElementById('device-modelid').value),
        name: document.getElementById('device-name').value,
        ip_address: document.getElementById('device-ip').value,
        serial_number: document.getElementById('device-sn').value
    };

    apiRequest('POST', '/devices', data).then(() => {
        showToast('创建设备成功', 'success');
        closeModal('modal-device');
        loadDevices();
    }).catch(error => showToast(error.message, 'error'));
}

function deleteDevice(id) {
    if (!confirm('确定要删除此设备吗？')) return;
    apiRequest('DELETE', `/devices/${id}`).then(() => {
        showToast('删除成功', 'success');
        loadDevices();
    }).catch(error => showToast(error.message, 'error'));
}

// ============ 固件管理 ============

async function loadFirmware() {
    try {
        const result = await apiRequest('GET', '/firmware?page=1&page_size=100');
        const list = result.data.list || [];
        const tbody = document.querySelector('#table-firmware tbody');

        if (list.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="loading">暂无固件</td></tr>';
            return;
        }

        tbody.innerHTML = list.map(f => `
            <tr>
                <td>${f.id}</td>
                <td>${f.model_name || '-'}</td>
                <td>${f.version}</td>
                <td>${formatFileSize(f.size)}</td>
                <td>${f.md5 || '-'}</td>
                <td>${f.download_count || 0}</td>
                <td>${new Date(f.release_date).toLocaleDateString()}</td>
                <td>
                    <div class="action-btns">
                        <a href="/api/firmware/${f.id}/download" class="btn btn-sm btn-secondary" download>下载</a>
                        <button class="btn btn-sm btn-danger" onclick="deleteFirmware(${f.id})">删除</button>
                    </div>
                </td>
            </tr>
        `).join('');
    } catch (error) {
        showToast('加载固件列表失败: ' + error.message, 'error');
    }
}

function submitFirmware(event) {
    event.preventDefault();
    
    const formData = new FormData();
    formData.append('model_id', document.getElementById('firmware-modelid').value);
    formData.append('version', document.getElementById('firmware-version').value);
    formData.append('md5', document.getElementById('firmware-md5').value);
    formData.append('changelog', document.getElementById('firmware-changelog').value);
    formData.append('file', document.getElementById('firmware-file').files[0]);

    fetch('/api/firmware', {
        method: 'POST',
        body: formData
    }).then(response => {
        if (!response.ok) {
            throw new Error('上传失败');
        }
        showToast('上传成功', 'success');
        closeModal('modal-firmware');
        loadFirmware();
    }).catch(error => showToast(error.message, 'error'));
}

function deleteFirmware(id) {
    if (!confirm('确定要删除此固件吗？')) return;
    apiRequest('DELETE', `/firmware/${id}`).then(() => {
        showToast('删除成功', 'success');
        loadFirmware();
    }).catch(error => showToast(error.message, 'error'));
}

// ============ 升级任务 ============

async function loadTasks() {
    try {
        const result = await apiRequest('GET', '/tasks?page=1&page_size=100');
        const list = result.data.list || [];
        const tbody = document.querySelector('#table-tasks tbody');

        if (list.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="loading">暂无任务</td></tr>';
            return;
        }

        tbody.innerHTML = list.map(t => `
            <tr>
                <td>${t.id}</td>
                <td>${t.name}</td>
                <td>${t.model_name || '-'}</td>
                <td>${t.firmware_version}</td>
                <td>${taskTypeLabel(t.task_type)}</td>
                <td>${getStatusBadge(t.status)}</td>
                <td>
                    <div class="progress-bar">
                        <div class="progress-bar-inner" style="width:${t.progress}%"></div>
                    </div>
                    <div class="progress-text">${t.progress}% (${t.success_count}/${t.total_devices})</div>
                </td>
                <td>
                    <div class="action-btns">
                        ${t.status === 'pending' ? `
                            <button class="btn btn-sm btn-primary" onclick="startTask(${t.id})">启动</button>
                            <button class="btn btn-sm btn-secondary" onclick="editTask(${t.id})">编辑</button>
                        ` : ''}
                        ${t.status === 'running' ? `
                            <button class="btn btn-sm btn-danger" onclick="cancelTask(${t.id})">取消</button>
                        ` : ''}
                        <button class="btn btn-sm btn-danger" onclick="deleteTask(${t.id})">删除</button>
                    </div>
                </td>
            </tr>
        `).join('');
    } catch (error) {
        showToast('加载任务列表失败: ' + error.message, 'error');
    }
}

function taskTypeLabel(type) {
    const map = {
        'full': '全量',
        'grayscale': '灰度',
        'targeted': '指定设备'
    };
    return map[type] || type;
}

function updateFirmwareSelect() {
    const modelId = document.getElementById('task-modelid').value;
    if (!modelId) {
        document.getElementById('task-firmwareid').innerHTML = '';
        return;
    }

    apiRequest('GET', `/firmware?model_id=${modelId}&page=1&page_size=100`)
        .then(result => {
            const list = result.data.list || [];
            const select = document.getElementById('task-firmwareid');
            select.innerHTML = list.map(f => `<option value="${f.id}">${f.version} (${formatFileSize(f.size)})</option>`).join('');
        })
        .catch(error => showToast(error.message, 'error'));
}

function updateTaskType() {
    const type = document.getElementById('task-type').value;
    document.getElementById('task-ratio-group').style.display = type === 'grayscale' ? 'block' : 'none';
    document.getElementById('task-target-group').style.display = type === 'targeted' ? 'block' : 'none';
}

function submitTask(event) {
    event.preventDefault();
    const data = {
        name: document.getElementById('task-name').value,
        model_id: parseInt(document.getElementById('task-modelid').value),
        firmware_id: parseInt(document.getElementById('task-firmwareid').value),
        task_type: document.getElementById('task-type').value,
        grayscale_ratio: parseFloat(document.getElementById('task-ratio').value) || 10,
        target_devices: document.getElementById('task-targets').value.split(',').map(s => s.trim()).filter(Boolean),
        description: document.getElementById('task-desc').value,
        created_by: 'admin'
    };

    apiRequest('POST', '/tasks', data).then(() => {
        showToast('创建任务成功', 'success');
        closeModal('modal-task');
        loadTasks();
    }).catch(error => showToast(error.message, 'error'));
}

function startTask(id) {
    apiRequest('POST', `/tasks/${id}/start`).then(() => {
        showToast('任务已启动', 'success');
        loadTasks();
    }).catch(error => showToast(error.message, 'error'));
}

function cancelTask(id) {
    if (!confirm('确定要取消此任务吗？')) return;
    apiRequest('POST', `/tasks/${id}/cancel`).then(() => {
        showToast('任务已取消', 'success');
        loadTasks();
    }).catch(error => showToast(error.message, 'error'));
}

function deleteTask(id) {
    if (!confirm('确定要删除此任务吗？')) return;
    apiRequest('DELETE', `/tasks/${id}`).then(() => {
        showToast('删除成功', 'success');
        loadTasks();
    }).catch(error => showToast(error.message, 'error'));
}

function editTask(id) {
    apiRequest('GET', `/tasks/${id}`).then(result => {
        const t = result.data;
        document.getElementById('task-name').value = t.name;
        document.getElementById('task-modelid').value = t.model_id;
        document.getElementById('task-type').value = t.task_type;
        document.getElementById('task-ratio').value = t.grayscale_ratio;
        document.getElementById('task-desc').value = t.description || '';
        updateTaskType();
        updateFirmwareSelect();
        openModal('modal-task');
    }).catch(error => showToast(error.message, 'error'));
}

// ============ 升级历史 ============

async function loadHistory() {
    try {
        const result = await apiRequest('GET', '/history?page=1&page_size=50');
        const list = result.data.list || [];
        const tbody = document.querySelector('#table-history tbody');

        if (list.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="loading">暂无历史记录</td></tr>';
            return;
        }

        tbody.innerHTML = list.map(r => `
            <tr>
                <td>${r.id}</td>
                <td>${r.device_id}</td>
                <td>${r.task_name || '-'}</td>
                <td>${r.from_version}</td>
                <td>${r.to_version}</td>
                <td>${getStatusBadge(r.status)}</td>
                <td>${r.progress}%</td>
                <td>${new Date(r.started_at).toLocaleString()}</td>
            </tr>
        `).join('');
    } catch (error) {
        showToast('加载历史记录失败: ' + error.message, 'error');
    }
}

// ============ 统计分析 ============

async function loadStats() {
    try {
        // 详细统计
        const statsResult = await apiRequest('GET', '/stats/statistics');
        const stats = statsResult.data;

        document.getElementById('stat-success-total').textContent = stats.success_rate ? 
            Math.round(stats.total_tasks * stats.success_rate / 100) : 0;
        document.getElementById('stat-fail-total').textContent = stats.total_tasks || 0;
        document.getElementById('stat-success-rate2').textContent = (stats.success_rate || 0).toFixed(1) + '%';

        // 今日记录
        const recentResult = await apiRequest('GET', '/history/recent?limit=100');
        document.getElementById('stat-today-records').textContent = (recentResult.data || []).length;

        // 设备状态图
        const devStatusResult = await apiRequest('GET', '/stats/devices');
        const devData = Object.entries(devStatusResult.data || {}).map(([name, count]) => ({ name: deviceStatusLabel(name), count }));
        renderBarChart('chart-device-status', devData);

        // 任务状态图
        const taskStatusResult = await apiRequest('GET', '/stats/tasks');
        const taskData = Object.entries(taskStatusResult.data || {}).map(([name, count]) => ({ name: taskStatusLabel(name), count }));
        renderBarChart('chart-task-status', taskData);
    } catch (error) {
        showToast('加载统计数据失败: ' + error.message, 'error');
    }
}

function deviceStatusLabel(status) {
    const map = { 'online': '在线', 'offline': '离线', 'upgrading': '升级中', 'error': '错误' };
    return map[status] || status;
}

function taskStatusLabel(status) {
    const map = { 'pending': '等待', 'running': '执行中', 'completed': '已完成', 'failed': '已失败', 'cancelled': '已取消' };
    return map[status] || status;
}

// ============ 系统状态 ============

function checkHealth() {
    const healthEl = document.getElementById('health-status');
    const detailsEl = document.getElementById('health-details');
    healthEl.innerHTML = '<div class="status-loading">检查中...</div>';
    detailsEl.innerHTML = '';

    Promise.all([
        fetch('/health').then(r => r.json()),
        fetch('/ready').then(r => r.json())
    ]).then(([health, ready]) => {
        healthEl.innerHTML = `
            <div style="font-size:18px;font-weight:600;color:${health.code === 0 ? '#67c23a' : '#f56c6c'}">
                ${health.code === 0 ? '✅ 服务正常' : '❌ 服务异常'}
            </div>
            <div style="margin-top:8px;color:#909399">版本: ${health.data.version} | 运行时间: ${health.data.uptime}</div>
        `;
        detailsEl.innerHTML = `<h4 style="margin-bottom:10px">依赖状态:</h4><pre>${JSON.stringify(ready.data.dependencies, null, 2)}</pre>`;
    }).catch(() => {
        healthEl.innerHTML = '<div style="color:#f56c6c">❌ 无法连接到服务</div>';
    });
}

async function loadHealth() {
    checkHealth();
}

// ============ 初始化 ============

async function initSampleData() {
    if (!confirm('确定要初始化示例数据吗？这将创建测试数据。')) return;
    try {
        await apiRequest('POST', '/init/sample');
        showToast('示例数据初始化成功', 'success');
        loadDashboard();
    } catch (error) {
        showToast('初始化失败: ' + error.message, 'error');
    }
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', () => {
    // 处理初始路由
    handleRoute();
    window.addEventListener('hashchange', handleRoute);

    // 加载表单选项
    loadDeviceFormOptions();
});
