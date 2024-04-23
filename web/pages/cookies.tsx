import TopBar from "../components/landing/top-bar";
import Navbar from "../components/navbar";
import IconHome from "@mui/icons-material/Home";
import Typography from "@mui/material/Typography";
import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import Grid from "@mui/material/Grid";

const drawerItems = [
    {
        text: "Home",
        icon: <IconHome />,
        action: {
            path: "/",
        }
    },
  ];

export default function Cookies() {
    return <>
        <TopBar shadow={false} drawerItems={drawerItems}/>
        <Container
            component="main"
            maxWidth={false}
            sx={{
                backgroundColor: "var(--color-light)",
                paddingY: 3,
                height: "96vh"
            }}
        >
            <CssBaseline />
            <Grid
                container
                direction="column"
                justifyContent="center"
                alignItems="center"
            >
                <Typography
                    marginBottom="10px"
                    fontSize="1.5em"
                >Cookie Policy</Typography>
                
                <Typography
                    maxWidth="600px"
                >
                    MailboxSweep.com uses cookies to improve your experience, and to help us understand and
                    improve our site. Session cookies are used to securely log you in, and third-party cookies
                    may be used by Google Tag Manager to understand how users interact with the site. Additionally,
                    cookies may be set by MailboxSweep and by Google when you log into your Google account.

                    MailboxSweep does not track your activity on other sites.
                </Typography>
            </Grid>
        </Container>
    </>
}