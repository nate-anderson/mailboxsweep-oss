import { useEffect, useState } from "react";
import { API } from "../../client";
import { UserAgreement } from "../../client-gen";
import Box from "@mui/material/Box";
import Container from '@mui/material/Container';
import CssBaseline from "@mui/material/CssBaseline";
import Typography from "@mui/material/Typography";
import Navbar from "../../components/navbar";
import Card from "@mui/material/Card";
import Avatar from "@mui/material/Avatar";
import IconGrading from "@mui/icons-material/Grading";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Link from "next/link";
import IconCheckBoxOutlineBlank from "@mui/icons-material/CheckBoxOutlineBlank";
import IconCheckBox from '@mui/icons-material/CheckBox';

const renderAgreementListItem = (agr: UserAgreement) =>
    <Link
        href={"/agreement/" + agr.key}
        passHref
        style={{textDecoration: 'none', color: "black"}}
        title={
            agr.agreedAt
                ? "Agreed to on " + (agr.agreedAt as string)
                : "Agreement pending"
        }
    >
        <ListItem disablePadding>
            <ListItemButton sx={{
                marginY: 1,
            }} key={agr.key}>
                <ListItemIcon>
                    {Boolean(agr.agreedAt) ? <IconCheckBox /> : <IconCheckBoxOutlineBlank />}
                </ListItemIcon>
                <ListItemText primary={agr.name}></ListItemText>
            </ListItemButton>
        </ListItem>
    </Link>

export default function Agreements() {
    const [
        agreements,
        setAgreements
    ] = useState<UserAgreement[]>();
    const [ failureMessage, setFailureMessage ] = useState<string | undefined>();

    useEffect(() => {
        API.listUserAgreements()
            .then(res => {
                setAgreements(res.agreements);
            })
            .catch(err => {
                setFailureMessage("Failed to fetch your agreements, please try again later.");
            })
    }, []);

    return (
        <>
            <Navbar />
            <Container component="main">
                <CssBaseline />
                <Box sx={{
                    marginTop: 6,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center'
                }}>
                    <Avatar sx={{ m: 1, bgcolor: 'var(--color-accent)' }}>
                        <IconGrading />
                    </Avatar>
                    <Typography sx={{
                        color: "var(--color-black)",
                        fontSize: "2em",
                        fontWeight: 100,
                    }}>
                        Agreements
                    </Typography>
                </Box>
                <Card
                    sx={{
                        maxWidth: 600,
                        width: "auto",
                        marginTop: 3,
                        marginX: "auto",
                    }}
                >
                    <List>
                    {agreements?.map(renderAgreementListItem)}
                    </List>
                </Card>
            </Container>
        </>
        
    )
}