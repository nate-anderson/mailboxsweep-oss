import AppBar from "@mui/material/AppBar";
import ToolBar from "@mui/material/Toolbar";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Container from "@mui/material/Container";
import IconButton from "@mui/material/IconButton";
import MenuIcon from "@mui/icons-material/Menu";

export default function Navbar() {
    return (
        <Box sx={{ flexGrow: 1 }}>
            <AppBar sx={{
                backgroundColor: "var(--color-primary)",
            }} position="static">
                <Container maxWidth="xl">
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
                        <IconButton edge="start" color="inherit" aria-label="menu">
                          <MenuIcon />
                        </IconButton>
                    </ToolBar>
                </Container>
            </AppBar>
        </Box>
    );
}