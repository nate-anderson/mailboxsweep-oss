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

export interface Step {
    // short display title
    label: string;
    // url path to display when this step is reached
    path: string;
    slug: StepSlug;
    hide?: boolean;
};

type StepSlug = 
    "TERMS" | 
    "PROFILE" | 
    "PLAN" |
    "PROVIDER_SIGNIN" |
    "ORGANIZE";

const stepsIMAP: Step[] = [
    {
        label: "Terms & Conditions",
        path: "/agreement/terms-and-conditions",
        slug: "TERMS",
    },
    {
        label: "Complete Profile",
        path: "/complete-profile",
        slug: "PROFILE",
    },
    {
        label: "Select Plan",
        path: "/plan/select",
        slug: "PLAN",
    },
    {
        label: "Email Login",
        path: "/email-account/login",
        slug: "PROVIDER_SIGNIN",
    },
    {
        label: "Organize Messages",
        path: "/classify/configure",
        slug: "ORGANIZE",
        hide: true,
    }
];

const stepsGmail: Step[] = [
    {
        label: "Terms & Conditions",
        path: "/agreement/terms-and-conditions",
        slug: "TERMS",
    },
    {
        label: "Select Plan",
        path: "/plan/select",
        slug: "PLAN",
    },
    {
        label: "Organize Messages",
        path: "/classify/configure",
        slug: "ORGANIZE",
        hide: true,
    }
];

export function getSteps(): Step[] | undefined {
    const provider = getProvider();

    // this is ugly, JS should support multi variable switches
    if (provider === "GMAIL") {
        return stepsGmail;
    } else if (provider === "IMAP") {
        return stepsIMAP;
    } else {
        console.log(`cannot select a step for provider ${provider}`);
        return undefined;
    }
}

export function getStepIndex(): number | undefined {
    let step = localStorage.getItem(stepKey);
    if (step) {
        return parseInt(step);
    }
    return undefined;
}

export function nextStep(): string | undefined {
    let step = getStepIndex();
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