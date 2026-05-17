document.addEventListener('DOMContentLoaded', () => {
    // State
    let projects = [];
    let projectsInterval = null;
    let logsInterval = null;
    let currentLogsProjectId = null;
    let autoScrollLogs = true;
    
    // DOM Elements
    const projectsGrid = document.getElementById('projects-grid');
    const projectsEmpty = document.getElementById('projects-empty');
    const projectCardTemplate = document.getElementById('project-card-template');
    
    // Modals
    const projectModal = document.getElementById('project-modal');
    const logsModal = document.getElementById('logs-modal');
    const confirmModal = document.getElementById('confirm-modal');
    
    // Form Elements
    const projectForm = document.getElementById('project-form');
    const modalTitle = document.getElementById('modal-title');
    const formId = document.getElementById('form-id');
    const formRepo = document.getElementById('form-repo');
    const formBranch = document.getElementById('form-branch');
    const formTriggerType = document.getElementById('form-trigger-type');
    const formTriggerInterval = document.getElementById('form-trigger-interval');
    const formRegServer = document.getElementById('form-reg-server');
    const formRegUser = document.getElementById('form-reg-user');
    const formRegPassEnv = document.getElementById('form-reg-pass-env');
    const formGitUser = document.getElementById('form-git-user');
    const formGitPassEnv = document.getElementById('form-git-pass-env');
    
    // Logs Elements
    const logsContent = document.getElementById('logs-content');
    
    // Initialize
    fetchProjects();
    startProjectsPolling();
    
    // Event Listeners
    document.getElementById('btn-add-project').addEventListener('click', () => openProjectModal());
    
    // Form Submit
    projectForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const payload = {
            repo: formRepo.value,
            branch: formBranch.value,
            trigger: {
                type: formTriggerType.value,
                interval: formTriggerInterval.value
            }
        };
        
        // Optional sections
        if (formRegServer.value || formRegUser.value || formRegPassEnv.value) {
            payload.registry = {};
            if (formRegServer.value) payload.registry.server = formRegServer.value;
            if (formRegUser.value) payload.registry.username = formRegUser.value;
            if (formRegPassEnv.value) payload.registry.password_env = formRegPassEnv.value;
        }
        
        if (formGitUser.value || formGitPassEnv.value) {
            payload.git_auth = {};
            if (formGitUser.value) payload.git_auth.username = formGitUser.value;
            if (formGitPassEnv.value) payload.git_auth.password_env = formGitPassEnv.value;
        }
        
        const id = formId.value;
        const method = id ? 'PUT' : 'POST';
        const url = id ? `/api/projects/${id}` : '/api/projects';
        
        try {
            const btn = document.getElementById('btn-save-project');
            btn.disabled = true;
            btn.textContent = 'Saving...';
            
            const res = await fetch(url, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            
            if (res.ok) {
                closeModal(projectModal);
                fetchProjects(); // Immediate refresh
            } else {
                console.error('Failed to save project');
                alert('Failed to save project. Please check the inputs.');
            }
        } catch (error) {
            console.error('Error saving project:', error);
            alert('Error saving project.');
        } finally {
            const btn = document.getElementById('btn-save-project');
            btn.disabled = false;
            btn.textContent = 'Save Project';
        }
    });
    
    // Modal Close Listeners
    document.querySelectorAll('.modal-close-btn').forEach(btn => {
        btn.addEventListener('click', () => closeModal(projectModal));
    });
    
    document.getElementById('logs-close-btn').addEventListener('click', () => {
        closeModal(logsModal);
        stopLogsPolling();
    });
    
    document.getElementById('confirm-cancel-btn').addEventListener('click', () => closeModal(confirmModal));
    document.getElementById('confirm-close-btn').addEventListener('click', () => closeModal(confirmModal));
    
    // Logs Scroll Listener (to disable auto-scroll if user scrolls up)
    logsContent.addEventListener('scroll', () => {
        const isAtBottom = logsContent.scrollHeight - logsContent.scrollTop <= logsContent.clientHeight + 10;
        autoScrollLogs = isAtBottom;
    });
    
    // --- API & State Functions ---
    
    async function fetchProjects() {
        try {
            const res = await fetch('/api/projects');
            if (!res.ok) throw new Error('Network response was not ok');
            const data = await res.json();
            projects = data || [];
            renderProjects();
        } catch (error) {
            console.error('Failed to fetch projects:', error);
            // Optionally show error state in UI
        }
    }
    
    function startProjectsPolling() {
        if (!projectsInterval) {
            projectsInterval = setInterval(fetchProjects, 5000);
        }
    }
    
    // --- Render Functions ---
    
    function renderProjects() {
        projectsGrid.innerHTML = '';
        
        if (projects.length === 0) {
            projectsGrid.classList.add('hidden');
            projectsEmpty.classList.remove('hidden');
            return;
        }
        
        projectsGrid.classList.remove('hidden');
        projectsEmpty.classList.add('hidden');
        
        projects.forEach(project => {
            const clone = projectCardTemplate.content.cloneNode(true);
            
            // Set data
            clone.querySelector('.repo-name').textContent = extractRepoName(project.repo);
            clone.querySelector('.repo-name').title = project.repo;
            clone.querySelector('.branch-name').textContent = project.branch;
            clone.querySelector('.interval-time').textContent = project.trigger?.interval || 'manual';
            
            // Status Badge
            const statusBadge = clone.querySelector('.status-badge');
            const status = project.last_status || 'Pending';
            statusBadge.textContent = status;
            statusBadge.classList.add(`status-${status.toLowerCase()}`);
            
            // Event Listeners for buttons
            clone.querySelector('.btn-run').addEventListener('click', () => triggerBuild(project.id));
            clone.querySelector('.btn-logs').addEventListener('click', () => openLogsModal(project.id));
            clone.querySelector('.btn-edit').addEventListener('click', () => openProjectModal(project));
            clone.querySelector('.btn-delete-action').addEventListener('click', () => confirmDelete(project.id));
            
            projectsGrid.appendChild(clone);
        });
    }
    
    // --- Action Functions ---
    
    async function triggerBuild(id) {
        try {
            const res = await fetch(`/api/projects/${id}/trigger`, { method: 'POST' });
            if (res.ok) {
                // Optimistically update or just fetch immediately
                fetchProjects();
            }
        } catch (error) {
            console.error('Failed to trigger build:', error);
        }
    }
    
    function confirmDelete(id) {
        openModal(confirmModal);
        const okBtn = document.getElementById('confirm-ok-btn');
        
        // Remove old event listeners by cloning
        const newOkBtn = okBtn.cloneNode(true);
        okBtn.parentNode.replaceChild(newOkBtn, okBtn);
        
        newOkBtn.addEventListener('click', async () => {
            try {
                newOkBtn.disabled = true;
                newOkBtn.textContent = 'Deleting...';
                const res = await fetch(`/api/projects/${id}`, { method: 'DELETE' });
                if (res.ok) {
                    closeModal(confirmModal);
                    fetchProjects();
                }
            } catch (error) {
                console.error('Failed to delete:', error);
            } finally {
                newOkBtn.disabled = false;
                newOkBtn.textContent = 'Delete';
            }
        });
    }
    
    // --- Logs Functions ---
    
    function openLogsModal(id) {
        currentLogsProjectId = id;
        logsContent.textContent = 'Loading logs...';
        autoScrollLogs = true;
        openModal(logsModal);
        
        fetchLogs();
        startLogsPolling();
    }
    
    async function fetchLogs() {
        if (!currentLogsProjectId) return;
        
        try {
            const res = await fetch(`/api/projects/${currentLogsProjectId}/logs`);
            if (res.ok) {
                const text = await res.text();
                // Only update if text has changed to avoid selection loss if user is highlighting
                if (logsContent.textContent !== text) {
                    logsContent.textContent = text || 'No logs available yet.';
                    if (autoScrollLogs) {
                        logsContent.scrollTop = logsContent.scrollHeight;
                    }
                }
            }
        } catch (error) {
            console.error('Failed to fetch logs:', error);
        }
    }
    
    function startLogsPolling() {
        stopLogsPolling(); // Ensure no duplicates
        logsInterval = setInterval(fetchLogs, 500);
    }
    
    function stopLogsPolling() {
        if (logsInterval) {
            clearInterval(logsInterval);
            logsInterval = null;
        }
        currentLogsProjectId = null;
    }
    
    // --- UI Utilities ---
    
    function openProjectModal(project = null) {
        projectForm.reset();
        
        if (project) {
            modalTitle.textContent = 'Edit Project';
            formId.value = project.id;
            formRepo.value = project.repo;
            formBranch.value = project.branch;
            formTriggerType.value = project.trigger?.type || 'polling';
            formTriggerInterval.value = project.trigger?.interval || '1m';
            
            if (project.registry) {
                formRegServer.value = project.registry.server || '';
                formRegUser.value = project.registry.username || '';
                formRegPassEnv.value = project.registry.password_env || '';
            }
            
            if (project.git_auth) {
                formGitUser.value = project.git_auth.username || '';
                formGitPassEnv.value = project.git_auth.password_env || '';
            }
        } else {
            modalTitle.textContent = 'Add Project';
            formId.value = '';
        }
        
        openModal(projectModal);
    }
    
    function openModal(modal) {
        modal.classList.remove('hidden');
        // Prevent body scrolling
        document.body.style.overflow = 'hidden';
    }
    
    function closeModal(modal) {
        modal.classList.add('hidden');
        // Restore body scrolling if no other modals are open
        if (!document.querySelectorAll('.modal-backdrop:not(.hidden)').length) {
            document.body.style.overflow = '';
        }
    }
    
    function extractRepoName(url) {
        if (!url) return 'Unknown Repo';
        // Extract "user/repo" from "https://github.com/user/repo.git"
        try {
            const parts = url.split('/');
            let name = parts[parts.length - 1];
            if (name.endsWith('.git')) {
                name = name.slice(0, -4);
            }
            // Include owner if available
            if (parts.length > 3) {
                const owner = parts[parts.length - 2];
                // basic heuristic
                if (!owner.includes('.') && !owner.includes(':')) {
                     return `${owner}/${name}`;
                }
            }
            return name;
        } catch (e) {
            return url;
        }
    }
});
