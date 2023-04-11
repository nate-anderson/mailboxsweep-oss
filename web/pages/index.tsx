import Head from "next/head";
import Mascot from "../components/Mascot";
import TopBar from "../components/landing/top-bar";
import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import Button from "@mui/material/Button";
import Grid from '@mui/material/Grid';
import Typography from "@mui/material/Typography";
import IconArrowForward from "@mui/icons-material/ArrowForward";

function CTA(props: { text: string, href: string }) {
  return <Button
    size="large"
    disableElevation={true}
    variant="contained"
    href={props.href}
    endIcon={<IconArrowForward />}
    sx={{
      backgroundColor: "var(--color-primary)",
      color: "white",
      ":hover": {
        backgroundColor: "var(--color-primary-hover)",
      },
      borderRadius: "0px",
      fontSize: "1em",
      textTransform: "none",
    }}
  >
    {props.text}
  </Button>
}

export default function Home() {
  return (
    <>
      <Head>
        <title>MailboxSweep cleans your inbox.</title>
        <meta name="description" content="Free up storage and declutter your digital life." />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <TopBar shadow={false}/>
      <Container
        component="main"
        maxWidth={false}
        sx={{
          backgroundColor: "var(--color-light)",
          height: "100vh",
          paddingY: 2,
        }}
      >
        <CssBaseline />
        <Grid
          container
          direction="row"
          justifyContent="center"
          alignItems="center"
        >
          <Grid
            container
            item
            xs={12}
            sm={12}
            md={8}
            lg={6}
            direction="row"
            justifyContent="center"
            alignItems="center"
          >
            <Grid
              item
              container
              xs={12}
              sm={12}
              md={6}
              direction="row"
              justifyContent="center"
              alignItems="center"
            >
              <Mascot />
            </Grid>
            <Grid
              item
              container
              xs={12}
              sm={12}
              md={6}
              direction="row"
              justifyContent="center"
              alignItems="center"
            >
              <Typography
                fontFamily="Roboto"
                fontSize="2.5em"
                sx={{
                  textAlign: {
                    xs: "center",
                    sm: "center",
                    md: "left",
                  },
                }}
              >
                Inbox cleanup made simple.
              </Typography>
              <Typography
                fontFamily="Roboto"
                color="var(--color-accent)"
                fontSize="1.25em"
                sx={{
                  marginY: 2,
                  textAlign: {
                    xs: "center",
                    sm: "center",
                    md: "left",
                  },
                }}
              >
                Free up space and declutter your digital life. We'll make it easy.
              </Typography>
            </Grid>
          </Grid>
          <Grid
            item
            xs={12}
            display="flex"
            justifyContent="center"
            sx={{
              marginY: 2
            }}
          >
            <CTA text="Start Cleaning Now" href="/login"/>
          </Grid>
        </Grid>
      </Container>
    </>
  );
}
