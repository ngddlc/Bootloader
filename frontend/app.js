const tasksList = document.getElementById("tasksList");
const emptyState = document.getElementById("emptyState");
const downloadForm = document.getElementById("downloadForm");
const urlInput = document.getElementById("urlInput");
const totalTasksCount = document.getElementById("totalTasksCount");

// Хранилище задач и состояния раскрытых контрольных сумм
const tasksMap = new Map();
const expandedChecksums = new Set();

        function toggleChecksum(taskId) {
            if (expandedChecksums.has(taskId)) {
                expandedChecksums.delete(taskId);
            } else {
                expandedChecksums.add(taskId);
            }
            const task = tasksMap.get(taskId);
            if (task) {
                renderTaskCard(task);
            }
        }

        // Форматирование байт в человеческий вид (КБ, МБ, ГБ)
        function formatBytes(bytes) {
            if (!bytes || bytes <= 0) return "0 Б";
            const k = 1024;
            const sizes = ["Б", "КБ", "МБ", "ГБ", "ТБ"];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
        }

        // Форматирование скорости
        function formatSpeed(bytesPerSec) {
            if (!bytesPerSec || bytesPerSec <= 0) return "--";
            if (bytesPerSec < 1024 * 1024) {
                return (bytesPerSec / 1024).toFixed(0) + " КБ/с";
            }
            return (bytesPerSec / (1024 * 1024)).toFixed(2) + " МБ/с";
        }

        // Возвращает бейдж категории
        function getCategoryBadge(category) {
            if (!category || category === "Другое") {
                return '<span class="px-2 py-0.5 text-xs font-medium bg-slate-800 text-slate-400 rounded-md border border-slate-700/80 flex items-center gap-1">📁 Другое</span>';
            }
            let icon = "📁";
            let style = "bg-slate-800 text-slate-300 border-slate-700";
            switch(category) {
                case "Архивы":
                    icon = "📦";
                    style = "bg-amber-500/10 text-amber-400 border-amber-500/20";
                    break;
                case "Документы":
                    icon = "📄";
                    style = "bg-blue-500/10 text-blue-400 border-blue-500/20";
                    break;
                case "Медиа":
                    icon = "🎬";
                    style = "bg-purple-500/10 text-purple-400 border-purple-500/20";
                    break;
                case "Программы":
                    icon = "⚙️";
                    style = "bg-emerald-500/10 text-emerald-400 border-emerald-500/20";
                    break;
            }
            return `<span class="px-2 py-0.5 text-xs font-medium rounded-md border ${style} flex items-center gap-1">${icon} ${category}</span>`;
        }

        // Возвращает стиль бейджа в зависимости от режима загрузки (Многопоточный vs Однопоточный)
        function getThreadsBadge(task) {
            if (!task.filename || task.filename === "probing...") {
                return '<span class="px-2 py-0.5 text-xs font-medium bg-slate-800 text-slate-400 rounded-md border border-slate-700/80 flex items-center gap-1.5" title="Сетевая разведка параметров сервера..."><span class="w-1.5 h-1.5 rounded-full bg-slate-400 animate-pulse"></span>Разведка...</span>';
            }
            const isMulti = task.is_multi_part !== undefined ? task.is_multi_part : (task.threads > 1);
            if (isMulti && task.threads > 1) {
                return `<span class="px-2 py-0.5 text-xs font-mono font-medium bg-emerald-500/10 text-emerald-400 rounded-md border border-emerald-500/20 flex items-center gap-1" title="Многопоточная загрузка: сервер поддерживает докачку частями (HTTP 206 Partial Content)">⚡ ${task.threads} потоков (Multi)</span>`;
            }
            return '<span class="px-2 py-0.5 text-xs font-mono font-medium bg-amber-500/10 text-amber-400 rounded-md border border-amber-500/20 flex items-center gap-1" title="Однопоточная загрузка: сервер не поддерживает Range-запросы (HTTP 200 OK)">🧵 1 поток (Single)</span>';
        }

        // Возвращает стиль бейджа в зависимости от статуса
        function getStatusBadge(status) {
            switch(status) {
                case "DOWNLOADING":
                    return '<span class="px-2.5 py-1 text-xs font-medium bg-blue-500/10 text-blue-400 rounded-lg border border-blue-500/20 flex items-center gap-1.5"><span class="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse"></span>Качается</span>';
                case "COMPLETED":
                    return '<span class="px-2.5 py-1 text-xs font-medium bg-emerald-500/10 text-emerald-400 rounded-lg border border-emerald-500/20">Готово</span>';
                case "CANCELED":
                    return '<span class="px-2.5 py-1 text-xs font-medium bg-amber-500/10 text-amber-400 rounded-lg border border-amber-500/20">Отменено</span>';
                case "FAILED":
                    return '<span class="px-2.5 py-1 text-xs font-medium bg-rose-500/10 text-rose-400 rounded-lg border border-rose-500/20">Ошибка</span>';
                default:
                    return `<span class="px-2.5 py-1 text-xs font-medium bg-slate-800 text-slate-400 rounded-lg">${status}</span>`;
            }
        }

        // Отрисовка или обновление карточки задачи
        function renderTaskCard(task) {
            tasksMap.set(task.id, task);
            totalTasksCount.innerText = tasksMap.size;
            emptyState.classList.add("hidden");

            let card = document.getElementById(`task-${task.id}`);
            
            // Извлекаем поля прогресса с поддержкой любого регистра
            const p = task.progress || {};
            const percentVal = p.percent !== undefined ? p.percent : (p.Percent !== undefined ? p.Percent : 0);
            const downloadedBytes = p.downloaded_bytes !== undefined ? p.downloaded_bytes : (p.DownloadedBytes !== undefined ? p.DownloadedBytes : 0);
            const totalBytes = (task.total_size && task.total_size > 0) ? task.total_size : (p.total_bytes || p.TotalBytes || 0);
            const speedVal = p.speed !== undefined ? p.speed : (p.Speed !== undefined ? p.Speed : 0);

            const percent = percentVal.toFixed(1);
            const downloaded = formatBytes(downloadedBytes);
            const total = totalBytes > 0 ? formatBytes(totalBytes) : "Определение...";
            const speed = formatSpeed(speedVal);
            const displayName = (!task.filename || task.filename === "probing...") ? "Определение файла..." : task.filename;

            const isDownloading = task.status === "DOWNLOADING";
            const isCompleted = task.status === "COMPLETED";
            const isChecksumExpanded = expandedChecksums.has(task.id);

            const cardHTML = `
                <div class="flex items-start justify-between gap-4 mb-2">
                    <div class="flex items-center gap-3 overflow-hidden">
                        <div class="p-2.5 bg-slate-800 rounded-xl text-slate-300 shrink-0">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                            </svg>
                        </div>
                        <div class="overflow-hidden">
                            <div class="flex items-center gap-2 flex-wrap">
                                <h3 class="font-medium text-slate-200 text-sm truncate" title="${displayName}">${displayName}</h3>
                                ${task.category ? getCategoryBadge(task.category) : ''}
                                ${getThreadsBadge(task)}
                                ${task.duplicate_of ? `<span class="px-2 py-0.5 text-xs font-medium bg-amber-500/10 text-amber-300 rounded-md border border-amber-500/30 flex items-center gap-1" title="Обнаружен дубликат файла: ${task.duplicate_of}">⚠️ Дубликат: ${task.duplicate_of}</span>` : ''}
                            </div>
                            <p class="text-xs text-slate-500 truncate" title="${task.url}">${task.url}</p>
                        </div>
                    </div>
                    <div class="flex items-center gap-2 shrink-0">
                        ${getStatusBadge(task.status)}
                        ${isDownloading ? `
                            <button onclick="cancelTask('${task.id}')" title="Отменить загрузку"
                                    class="p-1.5 hover:bg-rose-500/20 text-slate-400 hover:text-rose-400 rounded-lg transition border border-transparent hover:border-rose-500/30">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                                </svg>
                            </button>
                        ` : ''}
                        ${isCompleted && task.checksum ? `
                            <button onclick="toggleChecksum('${task.id}')" title="Контрольная сумма SHA-256"
                                    class="p-1.5 rounded-lg transition border border-transparent ${isChecksumExpanded ? 'text-emerald-400 bg-emerald-500/15 border-emerald-500/30' : 'text-slate-400 hover:text-emerald-400 hover:bg-slate-800'}">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>
                                </svg>
                            </button>
                        ` : ''}
                        ${isCompleted && task.dest_path ? `
                            <button onclick="openFolder('${task.id}')" title="Показать в проводнике"
                                    class="p-1.5 hover:bg-blue-500/20 text-slate-400 hover:text-blue-400 rounded-lg transition border border-transparent hover:border-blue-500/30">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 19a2 2 0 01-2-2V7a2 2 0 012-2h4l2 2h4a2 2 0 012 2v1M5 19h14a2 2 0 002-2v-5a2 2 0 00-2-2H9a2 2 0 00-2 2v5a2 2 0 01-2 2z"/>
                                </svg>
                            </button>
                        ` : ''}
                        ${!isDownloading ? `
                            <button onclick="openDeleteModal('${task.id}')" title="Удалить задачу"
                                    class="p-1.5 hover:bg-rose-500/20 text-slate-400 hover:text-rose-400 rounded-lg transition border border-transparent hover:border-rose-500/30">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                                </svg>
                            </button>
                        ` : ''}
                    </div>
                </div>

                <!-- Полоса прогресса -->
                <div class="w-full bg-slate-950 rounded-full h-2 overflow-hidden border border-slate-800/80 mb-2">
                    <div class="progress-bar ${isCompleted ? 'bg-emerald-500' : 'bg-gradient-to-r from-blue-600 to-indigo-500'} h-full rounded-full" style="width: ${percent}%"></div>
                </div>

                <!-- Метаинформация (проценты, байты, скорость / кнопка раскрытия SHA-256) -->
                <div class="flex justify-between items-center text-xs text-slate-400">
                    <span>${percent}% • ${downloaded} из ${total}</span>
                    <div class="flex items-center gap-2">
                        ${isDownloading ? `<span class="font-mono text-slate-300 font-medium">⚡ ${speed}</span>` : ''}
                        ${isCompleted && task.checksum ? `
                            <button onclick="toggleChecksum('${task.id}')"
                                    class="text-xs font-medium flex items-center gap-1.5 transition py-0.5 px-2 rounded-lg border border-transparent hover:border-slate-700/80 ${isChecksumExpanded ? 'text-emerald-400 bg-emerald-500/10' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800'}"
                                    title="Показать / скрыть контрольную сумму SHA-256">
                                <span>🛡️ SHA-256</span>
                                <svg class="w-3.5 h-3.5 transition-transform duration-200 ${isChecksumExpanded ? 'rotate-180 text-emerald-400' : 'text-slate-500'}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                                </svg>
                            </button>
                        ` : ''}
                    </div>
                </div>

                ${isCompleted && task.checksum && isChecksumExpanded ? `
                    <!-- Выдвижное поле с контрольной суммой SHA-256 -->
                    <div class="mt-2.5 pt-2.5 border-t border-slate-800/80 flex items-center justify-between gap-2 text-xs text-slate-400 animate-fadeIn">
                        <div class="flex items-center gap-1.5 overflow-hidden font-mono" title="Контрольная сумма SHA-256: ${task.checksum}">
                            <span class="text-emerald-400 font-semibold shrink-0">🛡️ SHA-256:</span>
                            <span class="text-slate-300 truncate select-all">${task.checksum}</span>
                        </div>
                        <button onclick="copyChecksum('${task.checksum}', this)" title="Скопировать SHA-256 в буфер обмена"
                                class="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 active:scale-95 text-slate-300 hover:text-white rounded-lg text-xs transition border border-slate-700 flex items-center gap-1.5 shrink-0">
                            <svg class="w-3.5 h-3.5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                            </svg>
                            <span>Копировать</span>
                        </button>
                    </div>
                ` : ''}
            `;

            if (!card) {
                card = document.createElement("div");
                card.id = `task-${task.id}`;
                card.className = "bg-slate-900/40 border border-slate-800/80 hover:border-slate-700/80 p-4 rounded-2xl transition duration-200 shadow-md";
                card.innerHTML = cardHTML;
                tasksList.prepend(card);
            } else {
                card.innerHTML = cardHTML;
            }
        }

        // Открытие папки с файлом в проводнике
        async function openFolder(taskIdOrPath) {
            try {
                let filePath = taskIdOrPath;
                if (tasksMap.has(taskIdOrPath)) {
                    filePath = tasksMap.get(taskIdOrPath).dest_path;
                }
                if (!filePath) return;
                if (window.go && window.go.main && window.go.main.App) {
                    await window.go.main.App.OpenFileFolder(filePath);
                }
            } catch (err) {
                console.error("Ошибка открытия папки:", err);
            }
        }

        // Отмена задачи по кнопке
        async function cancelTask(id) {
            try {
                if (window.go && window.go.main && window.go.main.App) {
                    await window.go.main.App.CancelDownload(id);
                }
            } catch (err) {
                console.error("Ошибка отмены:", err);
            }
        }

        let pendingDeleteTaskId = null;
        const deleteModal = document.getElementById("deleteModal");
        const deleteModalFilename = document.getElementById("deleteModalFilename");

        // Открытие модального окна подтверждения удаления
        function openDeleteModal(taskId) {
            pendingDeleteTaskId = taskId;
            const task = tasksMap.get(taskId);
            if (task && deleteModalFilename) {
                deleteModalFilename.innerText = task.filename || "Файл";
            }
            if (deleteModal) {
                deleteModal.classList.remove("hidden");
            }
        }

        // Закрытие модального окна
        function closeDeleteModal() {
            pendingDeleteTaskId = null;
            if (deleteModal) {
                deleteModal.classList.add("hidden");
            }
        }

        // Подтверждение удаления задачи (deleteFile: false = только из списка, true = вместе с файлом)
        async function confirmDelete(deleteFile) {
            if (!pendingDeleteTaskId) return;
            const id = pendingDeleteTaskId;
            closeDeleteModal();

            try {
                if (window.go && window.go.main && window.go.main.App) {
                    await window.go.main.App.DeleteTask(id, deleteFile);
                    tasksMap.delete(id);
                    expandedChecksums.delete(id);
                    const card = document.getElementById(`task-${id}`);
                    if (card) card.remove();
                    totalTasksCount.innerText = tasksMap.size;
                    if (tasksMap.size === 0) {
                        emptyState.classList.remove("hidden");
                    }
                }
            } catch (err) {
                console.error("Ошибка удаления задачи:", err);
            }
        }

        // Закрытие модалки по клавише Escape и клику вне карточки
        window.addEventListener("keydown", (e) => {
            if (e.key === "Escape" && deleteModal && !deleteModal.classList.contains("hidden")) {
                closeDeleteModal();
            }
        });
        if (deleteModal) {
            deleteModal.addEventListener("click", (e) => {
                if (e.target === deleteModal) {
                    closeDeleteModal();
                }
            });
        }

        let selectedCustomFolder = "";
        const folderBtn = document.getElementById("folderBtn");
        const folderLabel = document.getElementById("folderLabel");

        if (folderBtn) {
            folderBtn.addEventListener("click", async () => {
                try {
                    if (window.go && window.go.main && window.go.main.App) {
                        const folder = await window.go.main.App.SelectDownloadFolder();
                        if (folder) {
                            selectedCustomFolder = folder;
                            const folderName = folder.split(/[\\/]/).filter(Boolean).pop() || folder;
                            folderLabel.innerText = "📁 " + folderName;
                            folderBtn.classList.add("border-blue-500", "text-blue-400");
                            folderBtn.title = "Выбрана папка: " + folder + " (нажмите для смены)";
                        }
                    }
                } catch (err) {
                    console.error("Ошибка выбора папки:", err);
                }
            });
        }

        // Копирование контрольной суммы SHA-256 в буфер обмена
        function copyChecksum(hash, btn) {
            if (!hash) return;
            navigator.clipboard.writeText(hash).then(() => {
                const orig = btn.innerHTML;
                btn.innerHTML = `<span class="text-emerald-400 font-medium">✓ Скопировано</span>`;
                setTimeout(() => {
                    btn.innerHTML = orig;
                }, 1500);
            }).catch(err => {
                console.error("Не удалось скопировать хеш:", err);
            });
        }

        // Отправка формы создания новой задачи
        downloadForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            const url = urlInput.value.trim();
            if (!url) return;

            const threadsSelect = document.getElementById("threadsSelect");
            const threads = threadsSelect ? parseInt(threadsSelect.value, 10) || 8 : 8;

            try {
                if (window.go && window.go.main && window.go.main.App) {
                    const task = await window.go.main.App.StartDownload(url, selectedCustomFolder, threads);
                    renderTaskCard(task);
                    urlInput.value = "";
                } else {
                    alert("Wails бэкенд ещё инициализируется...");
                }
            } catch (err) {
                alert("Ошибка запуска загрузки: " + err);
            }
        });

        // Загрузка начальных данных и подписка на события Wails
        window.addEventListener("DOMContentLoaded", async () => {
            // Ждём инициализации Wails Runtime
            const checkWails = setInterval(async () => {
                if (window.go && window.go.main && window.go.main.App && window.runtime) {
                    clearInterval(checkWails);

                    // 1. Загружаем историю всех задач из SQLite
                    try {
                        const tasks = await window.go.main.App.GetTasks();
                        if (tasks && tasks.length > 0) {
                            tasks.forEach(t => renderTaskCard(t));
                        } else {
                            emptyState.classList.remove("hidden");
                        }
                    } catch (e) {
                        console.error("Не удалось загрузить задачи:", e);
                    }

                    // 2. Слушаем живые события прогресса от Go!
                    window.runtime.EventsOn("task:progress", (task) => {
                        renderTaskCard(task);
                    });
                }
            }, 100);
        });
