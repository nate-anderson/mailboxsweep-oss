import { Avatar, Box, Typography } from "@mui/material";
import { SignupStepper } from "./signup-stepper";
import { useEffect, useState } from "react";
import { getStepIndex } from "../lib/steps";

interface PageHeaderProps {
    showStepper?: boolean;
    title: string;
    icon: JSX.Element;
}

export default function PageHeader(props: PageHeaderProps) {
    // if a signup step is set in local storage, show the stepper
    const [ stepIsSet, setStepIsSet ] = useState(false);

    useEffect(() => {
        const step = getStepIndex();
        setStepIsSet(step !== undefined);
    }, []);

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
            props.showStepper && stepIsSet
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