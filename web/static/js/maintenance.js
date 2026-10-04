let maintenanceGeneration = 0;
let pendingFallback = {};

function escapeHTML(value) {
    return String(value || "").replace(/[&<>"']/g, c => ({"&":"&amp;", "<":"&lt;", ">":"&gt;", '"':"&quot;", "'":"&#39;"}[c]));
}
function pendingUpdates() {
    try { return JSON.parse(localStorage.getItem("dock-pending-updates") || "{}"); }
    catch (_) { return pendingFallback; }
}
function rememberUpdate(name, job) {
    const pending = pendingUpdates();
    if (job) pending[name] = job; else delete pending[name];
    pendingFallback = pending;
    try { localStorage.setItem("dock-pending-updates", JSON.stringify(pending)); } catch (_) {}
}
async function fetchJSON(url, options = {}, timeout = 15000) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeout);
    try {
        const response = await fetch(url, {...options, signal: controller.signal, cache: "no-store"});
        const text = await response.text();
        let data;
        try { data = JSON.parse(text); } catch (_) {
            if (!response.ok) throw new Error("Connection interrupted while reading the result");
            throw new Error("Unable to read operation result");
        }
        if (!response.ok) {
            const error = new Error(data.error || "Request rejected");
            error.rejected = true;
            throw error;
        }
        return data;
    } finally { clearTimeout(timer); }
}

async function loadUpdates() {
    const root = document.getElementById("updates");
    if (!root) return;
    const generation = ++maintenanceGeneration;
    root.innerHTML = '<div class="berth maintenance-scanning"><div class="scan-spinner"></div><div class="berth-info"><div class="berth-title">Scanning images...</div><div class="berth-status">Checking applied images</div></div></div>';
    try {
        const data = await fetchJSON("/api/updates", {}, 200000);
        if (generation !== maintenanceGeneration) return;
        root.innerHTML = "";
        const pending = pendingUpdates();
        for (const item of data.containers) {
            const tracked = pending[item.name];
            let text = {current:"Current", error:"Unable to check", unsupported:"Unsupported"}[item.status] || "Update available";
            let action = "";
            if (tracked || (item.status === "update" && item.can_update)) {
                action = `<button type="button" class="update-button" data-name="${escapeHTML(item.name)}" data-action="update">Update</button>`;
            }
            if (item.status === "update" && !item.can_update) text = item.message || "Host update required";
            root.insertAdjacentHTML("beforeend", `<div class="berth"><div class="berth-info"><div class="berth-title">${escapeHTML(item.name)}</div></div><div class="maintenance-actions"><div class="berth-state status-${escapeHTML(item.status)}" title="${escapeHTML(item.message)}">${escapeHTML(text)}</div>${action}</div></div>`);
        }
        root.querySelectorAll(".update-button").forEach(button => {
            const name = button.dataset.name;
            const state = button.closest(".berth").querySelector(".berth-state");
            button.addEventListener("click", () => {
                if (button.dataset.action === "check") { loadUpdates(); return; }
                startUpdate(name, button, state, generation);
            });
            if (pending[name]) {
                markUpdating(button, state, "Checking operation result...");
                waitForUpdate(name, pending[name], button, state, generation);
            }
        });
    } catch (error) {
        if (generation !== maintenanceGeneration) return;
        root.innerHTML = `<div class="berth"><div class="berth-info"><div class="berth-title">Unable to check updates</div><div class="berth-status">${escapeHTML(error.message)}</div></div></div>`;
        // A disconnected proxy can recover while an accepted update continues.
        if (Object.keys(pendingUpdates()).length) setTimeout(() => { if (generation === maintenanceGeneration) loadUpdates(); }, 5000);
    }
}
function markUpdating(button, state, message) {
    button.disabled = true;
    button.classList.add("updating");
    button.innerHTML = '<span class="update-spinner"></span>';
    state.className = "berth-state status-checking";
    state.textContent = message;
}
async function startUpdate(name, button, state, generation) {
    const id = (globalThis.crypto && crypto.randomUUID) ? crypto.randomUUID() : Date.now().toString(36) + Math.random().toString(36).slice(2);
    let job = {id, started: Date.now()};
    // Keep the id before POST: an interrupted response must never cause a blind retry.
    rememberUpdate(name, job);
    markUpdating(button, state, "Updating...");
    try {
        const result = await fetchJSON("/api/update/" + encodeURIComponent(name) + "?job=" + encodeURIComponent(id), {method:"POST"});
        job.id = result.job;
        rememberUpdate(name, job);
    } catch (error) {
        if (error.rejected) {
            rememberUpdate(name, null);
            showUpdateError(button, state, error.message);
            return;
        }
        state.textContent = "Connection interrupted — checking result...";
    }
    waitForUpdate(name, job, button, state, generation);
}
async function waitForUpdate(name, job, button, state, generation, attempts = 0) {
    if (generation !== maintenanceGeneration || !button.isConnected) return;
    if (Date.now() - job.started > 20 * 60 * 1000) {
        unknownUpdate(name, button, state, "Result not confirmed. Check container state before another update.");
        return;
    }
    try {
        const data = await fetchJSON("/api/update/" + encodeURIComponent(name) + "/status?job=" + encodeURIComponent(job.id));
        if (generation !== maintenanceGeneration || !button.isConnected) return;
        if (data.status === "completed") {
            rememberUpdate(name, null);
            state.className = "berth-state status-current";
            state.textContent = "Updated";
            button.remove();
            return;
        }
        if (data.status === "error") {
            rememberUpdate(name, null);
            showUpdateError(button, state, data.error || "Update failed. Check service state.");
            return;
        }
        if (data.status === "unknown") {
            unknownUpdate(name, button, state, data.error || "Operation result is unknown.");
            return;
        }
        markUpdating(button, state, "Updating...");
    } catch (_) {
        markUpdating(button, state, "Connection interrupted — checking result...");
    }
    setTimeout(() => waitForUpdate(name, job, button, state, generation, attempts + 1), Math.min(10000, 2000 + attempts * 500));
}
function unknownUpdate(name, button, state, message) {
    rememberUpdate(name, null);
    button.disabled = false;
    button.classList.remove("updating");
    button.textContent = "Check";
    button.dataset.action = "check";
    state.className = "berth-state status-error";
    state.textContent = message;
}
function showUpdateError(button, state, message) {
    button.disabled = false;
    button.classList.remove("updating");
    button.textContent = "Retry";
    state.className = "berth-state status-error";
    state.textContent = message || "Update failed";
}

let cleanupVisible = false;

async function loadCleanup() {
    const generation = ++maintenanceGeneration;

    const root = document.getElementById("updates");

    root.innerHTML = `
        <div class="berth maintenance-scanning">
            <div class="scan-spinner"></div>

            <div class="berth-info">
                <div class="berth-title">Checking Docker storage...</div>
                <div class="berth-status">Reading cleanup status</div>
            </div>
        </div>
    `;

    try {

        const response = await fetch("/api/cleanup", {
            cache: "no-store"
        });

        if (!response.ok) {
            throw new Error("Unable to read cleanup status");
        }

        const data = await response.json();
        if (generation !== maintenanceGeneration) return;

        root.innerHTML = "";

        const rows = [
            ["Images", data.images],
            ["Containers", data.containers],
            ["Volumes", data.volumes],
            ["Build Cache", data.build_cache]
        ];

        for (const [name, value] of rows) {

            const parts = value.split("|");

            const total = parts[0] || "0";
            const active = parts[1] || "0";
            const size = parts[2] || "0B";
            const reclaimable = parts[3] || "0B";

            root.insertAdjacentHTML("beforeend", `
                <div class="berth">

                    <div class="berth-info">
                        <div class="berth-title">${name}</div>

                        <div class="berth-status">
                            ${total} total · ${active} active · ${size}
                        </div>
                    </div>

                    <div class="berth-state">
                        ${reclaimable} reclaimable
                    </div>

                </div>
            `);
        }

        root.insertAdjacentHTML("beforeend", `
            <div class="berth">

                <div class="berth-info">
                    <div class="berth-title">Unused Images</div>
                    <div class="berth-status">
                        ${data.unused_count} images
                    </div>
                </div>

                <div class="maintenance-actions">
                    <div class="berth-state">
                        ${data.unused_size}
                    </div>
                </div>

            </div>

            <div class="cleanup-footer">
                <button
                    type="button"
                    id="cleanup-button"
                    class="update-button">
                    CLEAN
                </button>
            </div>
        `);

        const cleanupButton =
            document.getElementById("cleanup-button");

        const buildCacheParts = data.build_cache.split("|");
        const buildCacheReclaimable =
            buildCacheParts[3] || "0B";

        const nothingToClean =
            data.unused_count === 0 &&
            buildCacheReclaimable.startsWith("0B");

        if (nothingToClean) {
            cleanupButton.disabled = true;
        }

        cleanupButton.addEventListener("click", async () => {

            if (!confirm(
                "Remove unused Docker images and reclaimable build cache?"
            )) {
                return;
            }

            cleanupButton.disabled = true;
            cleanupButton.innerHTML =
                '<span class="update-spinner"></span>';

            try {

                const response = await fetch("/api/cleanup", {
                    method: "POST",
                    cache: "no-store",
                    headers: {
                        "Accept": "application/json"
                    }
                });

                if (!response.ok) {
                    throw new Error("Cleanup failed");
                }

                const result = await response.json();

                if (result.status !== "completed") {
                    throw new Error("Cleanup did not complete");
                }

                await loadCleanup();

            } catch (e) {

                cleanupButton.disabled = false;
                cleanupButton.textContent = "RETRY";
            }
        });

    } catch (e) {
        if (generation !== maintenanceGeneration) return;

        root.innerHTML = `
            <div class="berth">
                <div class="berth-info">
                    <div class="berth-title">Unable to read cleanup status</div>
                    <div class="berth-status">${e}</div>
                </div>
            </div>
        `;
    }
}

const cleanupToggle = document.getElementById("cleanup-toggle");

if (cleanupToggle) {

    cleanupToggle.addEventListener("click", () => {

        cleanupVisible = !cleanupVisible;

        if (cleanupVisible) {
            loadCleanup();
        } else {
            loadUpdates();
        }
    });
}

loadUpdates();
