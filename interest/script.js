function showAlert(alert) {
    const e = document.getElementById(alert);
    e.style.display = "block";
}

function hideAlert(alert) {
    const e = document.getElementById(alert);
    e.style.display = "none";
}

const showFailure = () => showAlert("failureAlert");
const showSuccess = () => showAlert("successAlert");

let hasSubmitted = false;

async function submit() {
    const input = document.getElementById("emailInput");
    const email = input.value;
    if (!hasSubmitted) {
        const result = await fetch({
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                email: email,
            })
        })
        if (result.status != 200) {
            showFailure();
        }
    }
}