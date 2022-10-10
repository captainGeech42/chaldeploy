// global object to track necessary elements
ELEMS = {
    auth: document.getElementById("btn-authenticate"),
    create: document.getElementById("btn-create-instance"),
    extend: document.getElementById("btn-extend-instance"),
    destroy: document.getElementById("btn-destroy-instance"),
    authStatus: document.getElementById("span-auth-status"),
    instanceStatus: document.getElementById("span-instance-status"),
    rctfAuthUrlField: document.getElementById("ta-rctf-auth-url")
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
    fetch("/api/auth", {
        method: "POST",
        body: ELEMS.rctfAuthUrlField.value
    }).then(r => {
        if (r.status == 403) {
            statusError(ELEMS.authStatus, "Couldn't auth to rCTF, bad token/URL?");
        } else if (r.status >= 500) {
            statusError(ELEMS.authStatus, "Server error, contact an @Admin");
        } else {
            return r.text();
        }
    }).then(teamName => {
        statusSuccess(ELEMS.authStatus, `Authenticated as ${teamName}`);
    });
}

// Register all event handlers for DOM elements
function registerHandlers() {
    ELEMS.rctfAuthUrlField.oninput = onAuthFieldChange;
    ELEMS.auth.onclick = onAuthenticate;
}

// Make sure that each element was successfully identified in ELEMS
// Returns true if all valid, otherwise false
function validateElems() {
    return !Object.keys(ELEMS).some(k => ELEMS[k] === null);
}

////////////////////////////////////////////////////////////////////////////////

if (validateElems()) {
    registerHandlers();
} else {
    console.error("Couldn't map elements into object, did the HTML change?");
}