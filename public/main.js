/* public/main.js */
let editingId = null; // 更新モード判定用の変数
let currentProjects = []; // ★ 追加: ログ表示用にデータを保持しておく配列

async function fetchProjects() {
    try {
        const res = await fetch('/api/projects');
        const projects = await res.json();
        const tbody = document.getElementById('project-list');
        
        // 編集中のプロジェクトがある場合は、UIのチラつきを防ぐため再描画をスキップする
        if (editingId) return;

        currentProjects = projects || []; // データを保持
        tbody.innerHTML = '';

        if (currentProjects.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;">No projects monitored yet.</td></tr>';
            return;
        }

        currentProjects.forEach(p => {
            const repoName = p.repo.split('/').pop().replace('.git', '');
            const pJson = JSON.stringify(p).replace(/"/g, '&quot;');

            // ★ 追加: ステータスバッジの生成
            let statusBadge = `<span class="badge" style="color:var(--pico-muted-color); cursor:pointer;" onclick="showLog('${p.id}')"><i class="fa-solid fa-minus"></i> Pending</span>`;
            if (p.last_status === "Building") {
                statusBadge = `<span class="badge" style="color:#0277bd; cursor:pointer;" onclick="showLog('${p.id}')"><i class="fa-solid fa-rotate fa-spin"></i> Building</span>`;
            } else if (p.last_status === "Success") {
                statusBadge = `<span class="badge" style="color:#388e3c; cursor:pointer;" onclick="showLog('${p.id}')"><i class="fa-solid fa-check-circle"></i> Success</span>`;
            } else if (p.last_status === "Failed") {
                statusBadge = `<span class="badge" style="color:#d32f2f; cursor:pointer;" onclick="showLog('${p.id}')"><i class="fa-solid fa-xmark-circle"></i> Failed</span>`;
            }

            // ★ td class="actions" の中に Run ボタンを追加
            tbody.innerHTML += `
                <tr>
                    <td>
                        <strong>${repoName}</strong><br>
                        <small><a href="${p.repo}" target="_blank">${p.repo}</a></small>
                    </td>
                    <td><code>${p.branch}</code></td>
                    <td>${statusBadge}</td> <!-- ★ ステータスを表示 -->
                    <td><i class="fa-solid fa-stopwatch"></i> ${p.trigger.interval}</td>
                    <td><i class="fa-solid fa-box"></i> ${p.registry.server}</td>
                    <td class="actions">
                        <button class="outline" onclick="triggerBuild('${p.id}')" title="Force Build"><i class="fa-solid fa-play"></i> Run</button>
                        <button class="outline secondary" onclick="editProject('${pJson}')">Edit</button>
                        <button class="outline contrast" onclick="deleteProject('${p.id}')">Delete</button>
                    </td>
                </tr>
            `;
        });
    } catch (err) {
        console.error("Failed to fetch projects:", err);
    }
}

// ★ 追加: ログモーダルの表示・非表示コントロール
function showLog(id) {
    const p = currentProjects.find(x => x.id === id);
    if (!p) return;
    
    document.getElementById('log-modal-title').innerHTML = `<i class="fa-solid fa-terminal"></i> Logs: ${p.repo.split('/').pop()}`;
    document.getElementById('log-modal-content').innerText = p.last_log || "No logs available yet. Please trigger a build.";
    document.getElementById('log-modal').showModal();
}

function closeLogModal() {
    document.getElementById('log-modal').close();
}

// --- 削除機能 ---
async function deleteProject(id) {
    if (!confirm(`Are you sure you want to delete project: ${id}?\nPolling will stop immediately.`)) return;
    try {
        const res = await fetch(`/api/projects/${id}`, { method: 'DELETE' });
        if (res.ok) {
            if (editingId === id) cancelEdit(); // 編集中のものを消した場合はフォームリセット
            fetchProjects();
        } else {
            alert("Failed to delete project");
        }
    } catch(err) {
        alert("Error connecting to API");
    }
}

// 手動ビルドのトリガー機能
async function triggerBuild(id) {
    try {
        const res = await fetch(`/api/projects/${id}/trigger`, { method: 'POST' });
        if (res.ok) {
            alert("Build triggered successfully! Check server logs.");
        } else {
            const errText = await res.text();
            alert(`Failed to trigger build: ${errText}`);
        }
    } catch(err) {
        alert("Error connecting to API");
    }
}

// --- 編集モードに切り替え ---
function editProject(projectStr) {
    const p = JSON.parse(projectStr);
    editingId = p.id;
    
    document.getElementById('form-title').innerHTML = '<i class="fa-solid fa-pen"></i> Edit Project';
    document.getElementById('repo').value = p.repo;
    document.getElementById('branch').value = p.branch;
    document.getElementById('interval').value = p.trigger.interval;
    document.getElementById('registry').value = p.registry.server;
    
    document.getElementById('submit-btn').innerText = "Update Project";
    document.getElementById('cancel-btn').style.display = 'inline-block';
    
    // フォームまでスクロール
    document.getElementById('add-form').scrollIntoView({ behavior: 'smooth' });
}

// --- 編集モードキャンセル ---
function cancelEdit() {
    editingId = null;
    document.getElementById('add-form').reset();
    
    // 初期値に戻す
    document.getElementById('branch').value = 'main';
    document.getElementById('interval').value = '1m';
    document.getElementById('registry').value = 'localhost:5000';

    document.getElementById('form-title').innerHTML = '<i class="fa-solid fa-plus"></i> Add New Project';
    document.getElementById('submit-btn').innerText = "Start Monitoring";
    document.getElementById('cancel-btn').style.display = 'none';
}

// --- 送信処理 (POST / PUT 自動切り替え) ---
document.getElementById('add-form').addEventListener('submit', async (e) => {
    e.preventDefault(); 
    
    const payload = {
        repo: document.getElementById('repo').value,
        branch: document.getElementById('branch').value,
        trigger: { type: 'polling', interval: document.getElementById('interval').value },
        registry: { 
            server: document.getElementById('registry').value,
            username: '', password_env: '' 
        }
    };

    // editingId の有無で URL と HTTPメソッド を切り替え
    const method = editingId ? 'PUT' : 'POST';
    const url = editingId ? `/api/projects/${editingId}` : '/api/projects';

    try {
        const res = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        
        if (res.ok) {
            cancelEdit(); // 送信成功したらフォームをリセット
            fetchProjects(); 
        } else {
            const errText = await res.text();
            alert(`Failed to save! (${errText})`);
        }
    } catch (err) {
        alert("Error connecting to Stower API.");
    }
});

// 初回読み込み
fetchProjects();

// ★ 追加: 5秒ごとに自動リフレッシュしてステータスを監視する！
setInterval(fetchProjects, 5000);
