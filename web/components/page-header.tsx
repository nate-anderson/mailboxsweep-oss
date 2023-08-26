import { Avatar, Box, Typography } from "@mui/material";
import { SignupStepper } from "./signup-stepper";
import { useEffect, useState } from "react";
import { StepSlug, getStepIndex, getSteps } from "../lib/steps";

interface PageHeaderProps {
    showStepper?: boolean;
    stepperSlug?: StepSlug;
    title: string;
    icon: JSX.Element;
}

export default function PageHeader(props: PageHeaderProps) {
    // if a signup step is set in local storage, show the stepper
    const [ stepIsSet, setStepIsSet ] = useState(false);
    // if the provided slug is not in the user's signup path, don't show the stepper here
    const [ includeSlug, setIncludeSlug ] = useState(false);

    useEffect(() => {
        const step = getStepIndex();
        setStepIsSet(step !== undefined);
        const steps = getSteps();
        if (steps && steps.find((step) => step.slug === props.stepperSlug)) {
            setIncludeSlug(true);
        }
    }, [props.stepperSlug]);

    return <Box
        sx={{
            marginTop: 6,
            maxWidth: 600,
            width: "auto",
            marginX: "auto",
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
        }}
    >
        {
            props.showStepper && stepIsSet && includeSlug
                ? <SignupStepper />
                : <Avatar sx={{ m: 1, bgcolor: "var(--color-accent)" }}>
                    {props.icon}
                </Avatar>
        }
        
        <Typography
            align="center"
            sx={{
                color: "var(--color-black)",
                fontSize: "2em",
                fontWeight: 100,
            }}
        >
        {props.title}
        </Typography>
    </Box>
}