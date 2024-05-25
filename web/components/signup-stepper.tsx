import Stepper from "@mui/material/Stepper";
import Step from "@mui/material/Step";
import { Step as SignupStep, getStepIndex, getSteps, setStep } from "../lib/steps";
import StepLabel from "@mui/material/StepLabel";
import { useEffect, useState } from "react";

export function getInitialStepPath(hasAgreedToTerms: boolean, hasPaid: boolean, profileComplete: boolean): string | undefined {
    const steps = getSteps();
    if (!steps) {
        return undefined;
    }

    if (!hasPaid) {
        const paymentStep = steps.findIndex(step => step.slug === "CHECKOUT");
        if (paymentStep > -1) {
            setStep(paymentStep);
            return steps[paymentStep].path;
        }

    } else if (!hasAgreedToTerms) {
        return "/agreement/terms-and-conditions";
    }

    return undefined;
}

export function SignupStepper() {
    const [ steps, setSteps ] = useState<SignupStep[] | undefined>();
    const [ currentStepIndex, setCurrentStepIndex ] = useState(0);
    
    useEffect(() => {
        let sti = getStepIndex();
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

