import AppBar from "@mui/material/AppBar";
import ToolBar from "@mui/material/Toolbar";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Container from "@mui/material/Container";
import IconButton from "@mui/material/IconButton";
import IconMenu from "@mui/icons-material/Menu";
import Drawer from "@mui/material/Drawer";
import { useState } from "react";
import ListItem from "@mui/material/ListItem";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import IconAccountCircle from "@mui/icons-material/AccountCircle";
import IconEnvelope from "@mui/icons-material/Email";
import IconCreditCard from "@mui/icons-material/CreditCard";
import IconClose from "@mui/icons-material/Close";
import Link from "next/link";

const renderDrawerItem = (props: { text: string, icon: any, path: string }) =>
    <Link key={props.text} href={props.path} passHref style={{textDecoration: 'none', color: "black"}}>
        <ListItem sx={{
            marginTop: 0,
            marginRight: 5,
            color: "inherit"
        }}>
            <ListItemButton>
                <ListItemIcon sx={{ marginLeft: 0 }}>
                    {props.icon}
                </ListItemIcon>
                <ListItemText>{props.text}</ListItemText>
            </ListItemButton>
        </ListItem>
    </Link>;

const drawerItems = [
    { text: "My Profile", icon: <IconAccountCircle />, path: "/complete-profile" },
    { text: "Email Account", icon: <IconEnvelope />, path: "/email-account/login"},
    { text: "Payment & Plan", icon: <IconCreditCard />, path: "#" },
];

export default function Navbar() {
    const [ drawerOpen, setDrawerOpen ] = useState(false);

    return (
        <Box sx={{ flexGrow: 1 }}>
            <AppBar 
                sx={{
                    backgroundColor: "var(--color-primary)",
                }}
                position="static"
            >
                <Container maxWidth={false}>
                    <ToolBar disableGutters sx={{
                        justifyContent: "space-between",
                    }}>
                        <div style={{width: "24px"}}></div>
                        <Typography sx={{
                            fontFamily: "Lobster",
                            fontSize: "1.5em",
                            color: "white",
                            marginX: "auto",
                        }}>MailboxSweep</Typography>
                        <IconButton edge="start" color="inherit" aria-label="menu" onClick={() => setDrawerOpen(!drawerOpen)}>
                          <IconMenu />
                        </IconButton>
                    </ToolBar>
                </Container>
                <Drawer
                    anchor="right"
                    open={drawerOpen}
                    onClose={() => setDrawerOpen(false)}
                >
                    <Box
                        sx={{
                            padding: 0,
                            display: "flex",
                            justifyContent: "right"
                        }}
                    >
                        <IconButton
                            edge="start"
                            aria-label="close menu"
                            onClick={() => setDrawerOpen(false)}
                            sx={{
                                marginTop: 2,
                                marginBottom: 0,
                                marginRight: 2,
                            }}
                        >
                            <IconClose />
                        </IconButton>
                    </Box>
                    {drawerItems.map(renderDrawerItem)}
                </Drawer>
            </AppBar>
        </Box>
    );
}