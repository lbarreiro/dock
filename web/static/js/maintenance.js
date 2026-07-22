async function loadUpdates() {

    const root = document.getElementById("updates");

    root.innerHTML = `
        <div class="berth maintenance-scanning">
            <div class="scan-spinner"></div>

            <div class="berth-info">
                <div class="berth-title">Scanning images...</div>
                <div class="berth-status">Checking for image updates</div>
            </div>
        </div>
    `;

    try {

        const response = await fetch("/api/updates");
        const data = await response.json();

        root.innerHTML = "";

        for (const item of data.containers) {

            let text = "Checking";
            let css = "status-checking";
            let action = "";

            switch (item.status) {

                case "current":
                    text = "Current";
                    css = "status-current";
                    break;

                case "update":
                    text = "";
                    css = "status-update";

                    action = `
                        <button
                            type="button"
                            class="update-button"
                            data-name="${item.name}">
                            Update
                        </button>
                    `;
                    break;

                case "error":
                    text = "Error";
                    css = "status-error";
                    break;
            }

            root.insertAdjacentHTML("beforeend", `
                <div class="berth">

                    <div class="berth-info">
                        <div class="berth-title">${item.name}</div>
                    </div>

                    <div class="maintenance-actions">

                        <div class="berth-state ${css}">
                        ${text
                            ? `<span class="status-dot">&bull;</span> ${text}`
                            : ""
                        }
                    </div>

                        ${action}

                    </div>

                </div>
            `);
        }

        document.querySelectorAll(".update-button").forEach(button => {

            button.addEventListener("click", async () => {

                const name = button.dataset.name;
                const berth = button.closest(".berth");
                const state = berth.querySelector(".berth-state");

                // Primeiro confirmar que o backend aceitou o update.
                // O spinner só começa depois da resposta "started".
                button.disabled = true;

                try {

                    const response = await fetch(
                        "/api/update/" + encodeURIComponent(name),
                        {
                            method: "POST",
                            cache: "no-store",
                            headers: {
                                "Accept": "application/json"
                            }
                        }
                    );

                    if (!response.ok) {
                        throw new Error("Update request failed");
                    }

                    const result = await response.json();

                    if (
                        result.status !== "started" &&
                        result.status !== "running"
                    ) {
                        throw new Error("Update was not started");
                    }

                    // Backend confirmou: agora sim mostramos Updating.
                    button.classList.add("updating");
                    button.innerHTML =
                        '<span class="update-spinner"></span>';

                    state.className = "berth-state status-checking";
                    state.innerHTML =
                        '<span class="status-dot">&bull;</span> Updating...';

                    waitForUpdate(name, button, state, 0);

                } catch (e) {

                    showUpdateError(button, state);

                }

            });

        });

    } catch (e) {

        root.innerHTML = `
            <div class="berth">
                <div class="berth-info">
                    <div class="berth-title">Unable to check updates</div>
                    <div class="berth-status">${e}</div>
                </div>
            </div>
        `;
    }
}


async function waitForUpdate(name, button, state, attempts = 0) {

    try {

        const response = await fetch(
            "/api/update/" +
            encodeURIComponent(name) +
            "/status",
            {
                cache: "no-store"
            }
        );

        if (!response.ok) {
            throw new Error("Unable to read update status");
        }

        const data = await response.json();

        console.log(
            "Update status:",
            name,
            data.status,
            "attempt:",
            attempts
        );

        if (data.status === "completed") {

            state.className = "berth-state status-current";
            state.innerHTML =
                '<span class="status-dot">&bull;</span> Current';

            button.remove();

            return;
        }

        if (data.status === "error") {

            showUpdateError(button, state);

            return;
        }

        /*
         * Depois de o POST ter sido aceite, running/idle são
         * estados transitórios válidos. Continuamos a esperar.
         *
         * 1800 tentativas x 1 segundo = 30 minutos.
         */
        if (attempts >= 1800) {

            showUpdateError(button, state);

            return;
        }

        setTimeout(() => {
            waitForUpdate(
                name,
                button,
                state,
                attempts + 1
            );
        }, 1000);

    } catch (e) {

        console.error("Update polling error:", e);

        showUpdateError(button, state);
    }
}


function showUpdateError(button, state) {

    button.disabled = false;
    button.classList.remove("updating");
    button.textContent = "Retry";

    state.className = "berth-state status-error";
    state.innerHTML =
        '<span class="status-dot">&bull;</span> Error';
}



let cleanupVisible = false;

async function loadCleanup() {

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
