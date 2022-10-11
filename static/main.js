// global object to track necessary elements
ELEMS = {
    auth: document.getElementById("btn-authenticate"),
    create: document.getElementById("btn-create-instance"),
    extend: document.getElementById("btn-extend-instance"),
    destroy: document.getElementById("btn-destroy-instance"),
    authStatus: document.getElementById("span-auth-status"),
    instanceStatus: document.getElementById("span-instance-status"),
    rctfAuthUrlField: document.getElementById("ta-rctf-auth-url"),
    toastDiv: document.getElementById("notice-toast"),
    toastBody: document.getElementById("notice-toast-body")
}

// Enable a button to be clicked
function enableButton(btn) {
    if (btn.classList.contains("disabled")) {
        btn.classList.remove("disabled");
    }
}

// Disable a button from being clicked
function disableButton(btn) {
    if (!btn.classList.contains("disabled")) {
        btn.classList.add("disabled");
    }
}

// Set an informational status message
function statusInfo(span, text) {
    span.className = "status-info";
    span.innerText = text;
}

// Set a success status message
function statusSuccess(span, text) {
    span.className = "status-success";
    span.innerText = text;
}

// Set an error status message
function statusError(span, text) {
    span.className = "status-error";
    span.innerText = text;
}

// Set the proper enabled/disabled states on instance management buttons
function toggleStateButtons(isActive) {
    if (isActive) {
        disableButton(ELEMS.create);
        enableButton(ELEMS.extend);
        enableButton(ELEMS.destroy);
    } else {
        enableButton(ELEMS.create);
        disableButton(ELEMS.extend);
        disableButton(ELEMS.destroy);
    }
}

// Show a toast notification via Bootstrap
function showNotice(text) {
    toast = bootstrap.Toast.getOrCreateInstance(ELEMS.toastDiv);
    ELEMS.toastBody.innerText = text;

    toast.show();
}

// Handler for when the contents of the auth url field change
function onAuthFieldChange(e) {
    if (e?.target?.value?.length > 0) {
        enableButton(ELEMS.auth);
    } else {
        disableButton(ELEMS.auth);
    }
}

// Handler for the authenticate button being clicked
function onAuthenticate(e) {
    statusInfo(ELEMS.authStatus, "(attempting auth...)");

    fetch("/api/auth", {
        method: "POST",
        body: ELEMS.rctfAuthUrlField.value
    }).then(r => {
        if (r.status === 403) {
            statusError(ELEMS.authStatus, "Couldn't auth to rCTF, bad token/URL?");
        } else if (r.status >= 400) {
            statusError(ELEMS.authStatus, "Server error, contact an @Admin");
        } else {
            return r.text();
        }
    }).then(teamName => {
        if (teamName) {
            statusSuccess(ELEMS.authStatus, `Authenticated as ${teamName}`);
            disableButton(ELEMS.auth);
            ELEMS.rctfAuthUrlField.readOnly = true;

            getInstanceStatus();
        }
    });
}

// Get the current instance status from the server
// Enables buttons accordingly
function getInstanceStatus() {
    statusInfo(ELEMS.instanceStatus, "(fetching status...)")

    fetch("/api/status")
        .then(r => {
            if (r.status === 403) {
                statusError(ELEMS.authStatus, "Please refresh the page and re-authenticate");
            } else if (r.status >= 400) {
                statusError(ELEMS.instanceStatus, "Server error, contact an @Admin");
            } else {
                return r.json()
            }
        })
        .then(data => {
            if (data) {
                if (data?.state === "active") {
                    statusSuccess(ELEMS.instanceStatus, `Active instance available at ${data?.host}, expires at $TIME`);
                    toggleStateButtons(true);
                } else if (data?.state === "inactive") {
                    statusInfo(ELEMS.instanceStatus, "No active instance");
                    toggleStateButtons(false);
                } else {
                    statusError(ELEMS.instanceStatus, "Couldn't get instance info, contact an @Admin");
                    console.error(data);
                }
            }
        });
}

// Handler for the Create Instance button being clicked
function onCreate(e) {
    statusInfo(ELEMS.instanceStatus, "(creating instance...)")
    
    fetch("/api/create", { method: "POST" })
        .then(r => {
            if (r.status === 403) {
                statusError(ELEMS.authStatus, "Please refresh the page and re-authenticate");
            } else if (r.status >= 400) {
                statusError(ELEMS.instanceStatus, "Server error, contact an @Admin");
            } else {
                return r.json()
            }
        })
        .then(data => {
            if (data) {
                statusSuccess(ELEMS.instanceStatus, `Active instance available at ${data?.host}, expires at $TIME`);
                toggleStateButtons(true);
                showNotice("Successfully created instance");
            }
        });
}

// Handler for the Extend Instance button being clicked
function onExtend(e) {
    statusInfo(ELEMS.instanceStatus, "(extending instance...)")
    
    fetch("/api/extend", { method: "POST" })
        .then(r => {
            if (r.status === 403) {
                statusError(ELEMS.authStatus, "Please refresh the page and re-authenticate");
            } else if (r.status >= 400) {
                statusError(ELEMS.instanceStatus, "Server error, contact an @Admin");
            } else {
                return r.text();
            }
        })
        .then(data => {
            if (data) {
                showNotice("Successfully extended instance lifetime");
                getInstanceStatus();
            }
        });
}

// Handler for the Destroy Instance button being clicked
function onDestroy(e) {
    statusInfo(ELEMS.instanceStatus, "(destroying instance...)")
    
    fetch("/api/destroy", { method: "POST" })
        .then(r => {
            if (r.status === 403) {
                statusError(ELEMS.authStatus, "Please refresh the page and re-authenticate");
            } else if (r.status >= 400) {
                statusError(ELEMS.instanceStatus, "Server error, contact an @Admin");
            } else {
                showNotice("Successfully destroyed instance");
                getInstanceStatus();
            }
        });
}

// Register all event handlers for DOM elements
function registerHandlers() {
    ELEMS.rctfAuthUrlField.oninput = onAuthFieldChange;
    ELEMS.auth.onclick = onAuthenticate;
    ELEMS.create.onclick = onCreate;
    ELEMS.extend.onclick = onExtend;
    ELEMS.destroy.onclick = onDestroy;
}

// Make sure that each element was successfully identified in ELEMS
// Returns true if all valid, otherwise false
function validateElems() {
    return !Object.keys(ELEMS).some(k => ELEMS[k] === null);
}

////////////////////////////////////////////////////////////////////////////////

if (validateElems()) {
    registerHandlers();

    // on soft refresh, the old auth token may still be in the textarea
    // make it a little easier for the user to re-auth
    if (ELEMS.rctfAuthUrlField.value.length > 0) {
        enableButton(ELEMS.auth);
    }
} else {
    console.error("Couldn't map elements into object, did the HTML change?");
}
