let currentPage = 1;
const pageSize = 20;
let currentFilters = {
    status: '',
    type: ''
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadTasks();
    setupEventListeners();

    // Auto-refresh every 5 seconds
    setInterval(loadTasks, 5000);
});

function setupEventListeners() {
    document.getElementById('refreshBtn').addEventListener('click', loadTasks);
    document.getElementById('statusFilter').addEventListener('change', (e) => {
        currentFilters.status = e.target.value;
        currentPage = 1;
        loadTasks();
    });
    document.getElementById('typeFilter').addEventListener('change', (e) => {
        currentFilters.type = e.target.value;
        currentPage = 1;
        loadTasks();
    });
    document.getElementById('prevPage').addEventListener('click', () => {
        if (currentPage > 1) {
            currentPage--;
            loadTasks();
        }
    });
    document.getElementById('nextPage').addEventListener('click', () => {
        currentPage++;
        loadTasks();
    });

    // Modal
    const modal = document.getElementById('createTaskModal');
    const btn = document.getElementById('createTaskBtn');
    const span = document.getElementsByClassName('close')[0];

    btn.onclick = () => {
        modal.style.display = 'block';
        updatePayloadFields();
    };

    span.onclick = () => {
        modal.style.display = 'none';
    };

    window.onclick = (event) => {
        if (event.target == modal) {
            modal.style.display = 'none';
        }
    };

    document.getElementById('newTaskType').addEventListener('change', updatePayloadFields);
    document.getElementById('createTaskForm').addEventListener('submit', handleCreateTask);
}

async function loadTasks() {
    try {
        const params = new URLSearchParams({
            page: currentPage,
            page_size: pageSize,
            ...currentFilters
        });

        const response = await fetch(`/api/tasks?${params}`);
        const data = await response.json();

        renderTasks(data.tasks || []);
        updatePagination(data);
        updateStats(data.tasks || []);
    } catch (error) {
        console.error('Failed to load tasks:', error);
        document.getElementById('tasksBody').innerHTML = `
            <tr><td colspan="6" class="loading">Failed to load tasks</td></tr>
        `;
    }
}

function renderTasks(tasks) {
    const tbody = document.getElementById('tasksBody');

    if (tasks.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="loading">No tasks found</td></tr>';
        return;
    }

    tbody.innerHTML = tasks.map(task => `
        <tr>
            <td title="${task.id}">${task.id.substring(0, 8)}...</td>
            <td>${task.type}</td>
            <td><span class="status-badge status-${task.status}">${task.status}</span></td>
            <td>${task.attempts} / ${task.max_attempts}</td>
            <td>${formatDate(task.created_at)}</td>
            <td>${formatDate(task.updated_at)}</td>
        </tr>
    `).join('');
}

function updatePagination(data) {
    document.getElementById('pageInfo').textContent =
        `Page ${data.page} of ${data.total_pages || 1} (${data.total} total)`;
    document.getElementById('prevPage').disabled = data.page <= 1;
    document.getElementById('nextPage').disabled = data.page >= data.total_pages;
}

function updateStats(tasks) {
    const stats = tasks.reduce((acc, task) => {
        acc[task.status] = (acc[task.status] || 0) + 1;
        return acc;
    }, {});

    document.getElementById('totalTasks').textContent = tasks.length;
    document.getElementById('newTasks').textContent = stats.new || 0;
    document.getElementById('processingTasks').textContent = stats.processing || 0;
    document.getElementById('doneTasks').textContent = stats.done || 0;
}

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString();
}

function updatePayloadFields() {
    const taskType = document.getElementById('newTaskType').value;
    const container = document.getElementById('payloadFields');

    let fieldsHTML = '';

    switch (taskType) {
        case 'email':
            fieldsHTML = `
                <label>To Email: <input type="email" id="email_to" required></label>
                <label>Subject: <input type="text" id="email_subject" required></label>
                <label>Body: <textarea id="email_body" required></textarea></label>
            `;
            break;
        case 'notification':
            fieldsHTML = `
                <label>User ID: <input type="text" id="notif_user_id" required></label>
                <label>Title: <input type="text" id="notif_title" required></label>
                <label>Message: <textarea id="notif_message" required></textarea></label>
            `;
            break;
        case 'report':
            fieldsHTML = `
                <label>Report Type: <select id="report_type" required>
                    <option value="sales">Sales</option>
                    <option value="usage">Usage</option>
                    <option value="analytics">Analytics</option>
                </select></label>
                <label>Date From: <input type="date" id="report_date_from" required></label>
                <label>Date To: <input type="date" id="report_date_to" required></label>
                <label>Format: <select id="report_format" required>
                    <option value="pdf">PDF</option>
                    <option value="csv">CSV</option>
                    <option value="xlsx">XLSX</option>
                </select></label>
            `;
            break;
        case 'webhook':
            fieldsHTML = `
                <label>URL: <input type="url" id="webhook_url" required></label>
                <label>Method: <select id="webhook_method" required>
                    <option value="POST">POST</option>
                    <option value="PUT">PUT</option>
                </select></label>
                <label>Body (JSON): <textarea id="webhook_body" required></textarea></label>
            `;
            break;
    }

    container.innerHTML = fieldsHTML;
}

async function handleCreateTask(e) {
    e.preventDefault();

    const taskType = document.getElementById('newTaskType').value;
    let payload = {};

    switch (taskType) {
        case 'email':
            payload = {
                to: document.getElementById('email_to').value,
                subject: document.getElementById('email_subject').value,
                body: document.getElementById('email_body').value
            };
            break;
        case 'notification':
            payload = {
                user_id: document.getElementById('notif_user_id').value,
                title: document.getElementById('notif_title').value,
                message: document.getElementById('notif_message').value
            };
            break;
        case 'report':
            payload = {
                report_type: document.getElementById('report_type').value,
                date_from: document.getElementById('report_date_from').value,
                date_to: document.getElementById('report_date_to').value,
                format: document.getElementById('report_format').value
            };
            break;
        case 'webhook':
            payload = {
                url: document.getElementById('webhook_url').value,
                method: document.getElementById('webhook_method').value,
                headers: {
                    "Content-Type": "application/json"
                },
                body: document.getElementById('webhook_body').value
            };
            break;
    }

    try {
        const response = await fetch('/api/tasks', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                type: taskType,
                payload: payload
            })
        });

        if (response.ok) {
            document.getElementById('createTaskModal').style.display = 'none';
            document.getElementById('createTaskForm').reset();
            loadTasks();
        } else {
            alert('Failed to create task');
        }
    } catch (error) {
        console.error('Failed to create task:', error);
        alert('Failed to create task');
    }
}
