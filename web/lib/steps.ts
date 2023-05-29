export type Provider =
    "GMAIL" 
    | "IMAP";

export interface Step {
    // short display title
    label: string;
    // url path to display when this step is reached
    successPath: string;

    next?: Step;
};

export const stepVariants: Record<Provider, Record<string, Step>> = {
    "GMAIL": {
        initial: {
            label: "Terms & Conditions",
            successPath: "/plan/select",
            next: {
                label: "Select a Plan",
                successPath: "/email-account/login",
                next: {
                    label: "Email Provider Login",
                    successPath: "/classify/configure"
                }
            }
        },
    },
    "IMAP": {
        initial: {
            label: "Terms & Conditions",
            successPath: "/plan/select",
            next: {
                label: "Select a Plan",
                successPath: "/classify/configure"
            }
        },
    }
}

function randomObjectProperty<valueT extends any>(object: Record<string, valueT>): valueT {
    const keys = Object.keys(object);
    return object[keys[ keys.length * Math.random() << 0]];
}

const stepKey = "mailboxsweep_signup_step";

export function setStep(step: Step) {
    localStorage.setItem(stepKey, JSON.stringify(step));
}

function pickStepVariant(provider: Provider): Step {
    return randomObjectProperty(stepVariants[provider]);
}

// export function getStep(): Step | undefined {
//     let step = localStorage.getItem(stepKey);
//     if (step) {
//         return JSON.parse(step) as Step;
//     }

//     step = pickStepVariant
// }
