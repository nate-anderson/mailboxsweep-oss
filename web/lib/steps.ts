export type Provider =
    "GMAIL" 
    | "IMAP";

export interface Step {
    // short display title
    label: string;
    // url path to display when this step is reached
    path: string;
    next?: Step;
};

const providerKey = "mailboxsweep_provider";
const stepKey = "mailboxsweep_signup_step";
const StepsComplete = "COMPLETE";

export function setStep(index: number | undefined) {
    if (!index) {
        localStorage.removeItem(stepKey);
        return;
    }

    localStorage.setItem(stepKey, index.toString());
}

export function setProvider(provider: Provider) {
    localStorage.setItem(providerKey, provider);
}

export function getProvider(): Provider | undefined {
    const provider = localStorage.getItem(providerKey);
    if (provider == "GMAIL" || provider == "IMAP") {
        return provider;
    }
    return undefined;
}

export function getStep(): number | undefined {
    let step = localStorage.getItem(stepKey);
    if (step) {
        return parseInt(step);
    }
    return undefined;
}

export function nextStep(): string | undefined {
    let step = getStep();
    if (step) {
        step++;
        setStep(step);
    }
    return undefined;
}

export function getProviderByAddress(address: string): Provider {
    const host = address.split("@")[1];
    switch (host.toLowerCase()) {
        case "gmail.com":
            return "GMAIL";
        default:
            return "IMAP";
    }
}