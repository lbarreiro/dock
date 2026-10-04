if (window.lucide) lucide.createIcons();

let last = {
    cpu: "",
    ram: "",
    temp: "",
    running: "",
    uptime: ""
};

async function refreshSystem() {

    try {

        const r = await fetch("/api/system");
        const s = await r.json();

        const cpu = `${Math.round(s.cpu)}%`;
        const ram = `${Math.round(s.ram)}%`;
        const temp = `${Math.round(s.temp)}°C`;
        const running = `${s.running}/${s.total}`;
        const uptime = `${s.uptime}`;

        if (cpu !== last.cpu) {
            document.querySelector("#cpu .sys-value").textContent = cpu;
            last.cpu = cpu;
        }

        if (ram !== last.ram) {
            document.querySelector("#ram .sys-value").textContent = ram;
            last.ram = ram;
        }

        if (temp !== last.temp) {
            document.querySelector("#temp .sys-value").textContent = temp;
            last.temp = temp;
        }

        if (running !== last.running) {
            document.getElementById("running").innerHTML = `<i data-lucide="boxes"></i> RUN <span class="sys-value">${running}</span>`;
            if (window.lucide) lucide.createIcons();
            last.running = running;
        }

        if (uptime !== last.uptime) {
            document.querySelector("#uptime .sys-value").textContent = uptime;
            last.uptime = uptime;
        }

    } catch (_) {}

}

async function toggleContainer(id) {

    const button = document.getElementById("toggle-" + id);

    const original = button.innerHTML;

    button.disabled = true;
    button.innerHTML = '<div class="spinner"></div>';

    try {

        const r = await fetch("/api/container/" + id + "/toggle", {
            method: "POST"
        });

        if (r.ok) {

            await new Promise(resolve => setTimeout(resolve, 1000));

            const rr = await fetch("/api/container/" + id);

            if (!rr.ok) {
                throw new Error("Unable to fetch container");
            }

            const c = await rr.json();

            // TODO: atualizar apenas a linha do container

            updateContainer(c);

            refreshSystem();

        } else {

            button.disabled = false;
            button.innerHTML = original;
            if (window.lucide) lucide.createIcons();

        }

    } catch (_) {

        button.disabled = false;
        button.innerHTML = original;
        if (window.lucide) lucide.createIcons();

    }

}


function updateContainer(c) {

    const anchor = document.getElementById("anchor-" + c.id);
    const status = document.getElementById("status-" + c.id);
    const button = document.getElementById("toggle-" + c.id);

    if (!anchor || !status || !button) {
        return;
    }

    anchor.className = c.state === "running"
        ? "anchor running"
        : "anchor stopped";

    status.textContent = c.status;

    button.className = c.state === "running"
        ? "stop"
        : "start";

    button.title = c.state === "running"
        ? "Stop"
        : "Start";

    button.innerHTML = c.state === "running"
        ? '<i data-lucide="square"></i>'
        : '<i data-lucide="play"></i>';

    button.setAttribute(
        "onclick",
        "toggleContainer('" + c.id + "')"
    );

    button.disabled = false;

    if (window.lucide) lucide.createIcons();
}

async function openLogs(id) {

    const modal = document.getElementById("logsModal");
    const content = document.getElementById("logsContent");

    modal.classList.add("show");
    content.textContent = "Loading...";

    try {

        const r = await fetch("/api/container/" + id + "/logs");

        if (!r.ok) {
            content.textContent = "Unable to load logs.";
            return;
        }

        content.textContent = await r.text();
        content.scrollTop = content.scrollHeight;

    } catch (_) {

        content.textContent = "Unable to load logs.";

    }

}

function closeLogs() {

    document.getElementById("logsModal").classList.remove("show");

}

window.addEventListener("keydown", function(e){

    if(e.key === "Escape"){
        closeLogs();
    }

});

window.addEventListener("click", function(e){

    const modal = document.getElementById("logsModal");

    if(e.target === modal){
        closeLogs();
    }

});

refreshSystem();

setInterval(refreshSystem, 3000);


async function activateEmergencyMode() {
    const confirmed = window.confirm(
        "Ativar modo de emergência?\n\n" +
        "Todos os containers não essenciais serão parados. " +
        "Dock, cloudflared, ntfy e whatssend ficarão a funcionar."
    );

    if (!confirmed) {
        return;
    }

    const button = document.getElementById("emergency-action");
    const original = button ? button.innerHTML : "";

    if (button) {
        button.disabled = true;
        button.innerHTML = '<div class="spinner"></div>';
    }

    try {
        const response = await fetch("/api/emergency", { method: "POST" });
        const text = await response.text();
        let result;
        try { result = JSON.parse(text); } catch (_) { throw new Error(text || "Unable to confirm emergency result"); }

        if (!response.ok) {
            throw new Error(result.error || (result.errors || []).join("\n") || "Emergency mode failed");
        }

        let message =
            (result.status === "completed" ? "Modo de emergência ativo.\n\n" : "Modo de emergência parcialmente concluído.\n\n") +
            result.stopped + " container(s) parado(s).\n" +
            result.essential_running + " serviço(s) essencial(is) a funcionar.";

        if (result.errors && result.errors.length) {
            message += "\n\nAvisos:\n" + result.errors.join("\n");
        }

        window.alert(message);
        window.location.reload();
    } catch (error) {
        window.alert("Não foi possível confirmar a conclusão da emergência.\nVerifica o estado dos serviços antes de repetir.\n\n" + error.message);

        if (button) {
            button.disabled = false;
            button.innerHTML = original;
            if (window.lucide) lucide.createIcons();
        }
    }
}
