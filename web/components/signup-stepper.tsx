import Stepper from "@mui/material/Stepper";
import Step from "@mui/material/Step";
import { getProvider, getStep } from "../lib/steps";
import StepLabel from "@mui/material/StepLabel";
import { useEffect, useState } from "react";

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

export function getInitialStepPath(hasAgreedToTerms: boolean, hasPaid: boolean, profileComplete: boolean): string | undefined {
    const steps = getSteps();
    if (!steps) {
        return undefined;
    }

    if (!hasAgreedToTerms) {
        const termsStep = steps.find(step => step.slug === "TERMS");
        if (termsStep) {
            return termsStep.path;
        }
    }

    if (!profileComplete) {
        const profileStep = steps.find(step => step.slug === "PROFILE");
        if (profileStep) {
            return profileStep.path;
        }
    }

    if (!hasPaid) {
        const paymentStep = steps.find(step => step.slug === "PLAN");
        if (paymentStep) {
            return paymentStep.path;
        }

    }

    if (steps.length > 0) {
        return steps[0].path;
    }

    return undefined;
}

export function SignupStepper() {
    const [ steps, setSteps ] = useState<Step[] | undefined>();
    const [ currentStepIndex, setCurrentStepIndex ] = useState(0);
    
    useEffect(() => {
        let sti = getStep();
        if (sti) {
            setCurrentStepIndex(sti);
        }

        const allSteps = getSteps();
        if (!allSteps) {
            return;
        }

        setSteps(allSteps);
    }, [steps, setSteps, currentStepIndex, setCurrentStepIndex]);

    return <Stepper
        sx={{
            marginBottom: 1,
        }}
        alternativeLabel
    >
        {steps
            ?.filter(step => !step.hide)
            ?.map((step, i) => <Step key={step.label} completed={i < currentStepIndex} active={currentStepIndex === i}>
            <StepLabel>{step.label}</StepLabel>
        </Step>)}
    </Stepper>;
}

