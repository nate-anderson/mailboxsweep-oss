import Head from "next/head";
import Mascot from "../components/Mascot";
import TopBar from "../components/landing/top-bar";
import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import Button from "@mui/material/Button";
import Grid from '@mui/material/Grid';
import Typography from "@mui/material/Typography";
import IconArrowForward from "@mui/icons-material/ArrowForward";
import UIDemo from "../components/landing/ui-demo";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemAvatar from '@mui/material/ListItemAvatar';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Avatar from '@mui/material/Avatar';
import EnvelopeGraphic from "../public/email-icon.svg";
import Image from "next/image";
import Link from "@mui/material/Link";
import PlanCard from "../components/plan";
import { join } from "path";
import { promises as fs } from 'fs';
import { parse as parseYaml } from "yaml";
import { Plan } from "../client-gen";
import Box from "@mui/material/Box";

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

function TextCTA(props: { text: string, href: string }) {
  return <Link
      href={props.href}
      sx={{
        textDecoration: "none",
        color: "var(--color-primary)",
        textTransform: "none",
        display: "inline",
        fontSize: "1em",
      }}
    >
      {props.text}
      &nbsp;
      <IconArrowForward fontSize="inherit" />
    </Link>;
}

interface IPlan {
  display_name: string;
  description: string;
  price_in_cents: number;
  features: string[];
  enabled: boolean;
  allowed_scans: number;
  duration_days: number;
}

const convertPlan = (p: IPlan, key: string): Plan => {
  return {
    key,
    displayName: p.display_name,
    description: p.description,
    features: p.features,
    allowedScans: p.allowed_scans,
    priceInCents: p.price_in_cents,
    durationDays: p.duration_days,
  }
}

interface IStaticProps {
  plans: Record<string, IPlan>;
}

export async function getStaticProps(): Promise<{ props: IStaticProps }> {
  const plansPath = join(process.cwd(), "plans.yml");
  const plansData = await fs.readFile(plansPath, { encoding: "utf-8" });
  const plans = parseYaml(plansData);
  return { props: { plans } };
}

export default function Home(props: IStaticProps) {
  console.log(props);
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
          paddingY: 3,
        }}
      >
        <CssBaseline />
        {/* Row 1 */}
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
                sx={{
                  textAlign: {
                    xs: "center",
                    sm: "center",
                    md: "left",
                  },
                  fontSize: {
                    xs: "2.3em",
                    md: "2.5em",
                  },
                  lineHeight: "1.35",
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
              marginY: 4
            }}
          >
            <CTA text="Start Cleaning Now" href="/login"/>
          </Grid>
        </Grid>
            
        {/* Row 2 */}
        <Grid
          container
          direction="row"
          justifyContent="center"
          alignItems="center"
          marginTop="20px"
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
            spacing={5}
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
              <UIDemo />
            </Grid>
            <Grid
              item
              container
              xs={10}
              sm={8}
              md={6}
              direction="row"
              justifyContent="center"
              alignItems="center"
            >
              <List
                sx={{
                  listStyleType: 'disc',
                  pl: 4,
                  fontFamily: "Roboto"
                }}
              >
                <ListItem
                  sx={{ display: 'list-item' }}
                >
                  Mark messages read to hide notification badges, or delete them entirely to free up space and declutter.
                </ListItem>
                <ListItem
                  sx={{ display: 'list-item' }}
                >
                  Select large groups to delete quickly, or make precise selections to keep exactly what you want.
                </ListItem>
                <ListItem
                  sx={{ display: 'list-item' }}
                >
                  Customize how your inbox is organized, or use our recommended settings for easy cleanup.
                </ListItem>
              </List>
            </Grid>
          </Grid>
        </Grid>

        {/* Row 3 */}
        <Grid
          container
          direction="row"
          justifyContent="center"
          alignItems="center"
          marginTop={{
            xs: "40px",
            md: "60px",
          }}
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
              justifyContent={{
                xs: "center",
                sm: "center",
                md: "right",
              }}
              alignItems="center"
              order={{
                xs: 1,
                md: 0,
              }}
            >
              <Typography
                fontFamily="Roboto"
                sx={{
                  textAlign: {
                    xs: "center",
                    sm: "center",
                    md: "right",
                  },
                  fontSize: "2em",
                  lineHeight: "1.35",
                  marginTop: 3
                }}
              >
                Sick of the Red Badge?
              </Typography>
              <Typography
                fontFamily="Roboto"
                color="var(--color-accent)"
                fontSize="1em"
                sx={{
                  marginY: 2,
                  textAlign: {
                    xs: "center",
                    sm: "center",
                    md: "right",
                  },
                }}
              >
                You'll never read those messages, why waste time hunting them down and marking them read? MailboxSweep can help you shrink the badge faster.
                &nbsp;
                <TextCTA text="Get started now" href="/login" />
              </Typography>
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
              order={{
                xs: 0,
                md: 1,
              }}
            >
              <Image src={EnvelopeGraphic}
                alt="A mobile email app icon with a badge indicating lots of unread messages"
                priority
                style={{
                  width: "30vw",
                  height: "auto",
                  maxWidth: "180",
                  marginRight: "-20px",
              }}
              />
            </Grid>
          </Grid>
        </Grid>

        {/* Row 4 */}
        <Grid
          container
          direction="row"
          justifyContent="center"
          alignItems="center"
          marginTop={{
            xs: "40px",
            md: "60px",
          }}
        >
          <Grid
            item
            xs={12}
            display="flex"
            justifyContent="center"
            sx={{
              marginBottom: 2,
            }}
          >
            <Typography
              fontSize="2em"
            >
              How it Works
            </Typography>
          </Grid>
          <Grid
              item
              container
              xs={10}
              sm={8}
              md={6}
              direction="row"
              justifyContent="center"
              alignItems="center"
            >
              <List
                sx={{
                  listStyleType: 'disc',
                  fontFamily: "Roboto",
                  marginBottom: 2,
                }}
              >
                <ListItem>
                  <ListItemAvatar>
                    <Avatar
                      sx={{
                        backgroundColor: "var(--color-accent)"
                      }}
                    >
                      1
                    </Avatar>
                  </ListItemAvatar>
                  <ListItemText
                    sx={{
                      paddingLeft: 3
                    }}
                  >
                    Create an account and sign into your email provider. Your information is stored with enterprise-grade encryption.
                  </ListItemText>
                </ListItem>
                <ListItem>
                  <ListItemAvatar>
                    <Avatar
                      sx={{
                        backgroundColor: "var(--color-accent)"
                      }}
                    >
                      2
                    </Avatar>
                  </ListItemAvatar>
                  <ListItemText
                    sx={{
                      paddingLeft: 3
                    }}
                  >
                    Choose how you want to classify your inbox messages, including by website, sender, date, and subject keywords.
                  </ListItemText>
                </ListItem>
                <ListItem>
                  <ListItemAvatar>
                    <Avatar
                      sx={{
                        backgroundColor: "var(--color-accent)"
                      }}
                    >
                      3
                    </Avatar>
                  </ListItemAvatar>
                  <ListItemText
                    sx={{
                      paddingLeft: 3
                    }}
                  >
                    Choose groups of messages to delete or mark read.
                  </ListItemText>
                </ListItem>
              </List>
            </Grid>
            <CTA text="Sign in and get started" href="/login" />
        </Grid>

        {/* Row 5 */}
        <Grid
          container
          direction="row"
          justifyContent="center"
          alignItems="center"
          marginTop={{
            xs: "40px",
            md: "60px",
          }}
        >
          <Grid
            item
            xs={12}
            md={12}
            lg={8}
            display="flex"
            justifyContent="center"
            sx={{
              marginBottom: 2,
            }}
          >
            <Typography
              fontSize="2em"
            >
              Plans
            </Typography>
          </Grid>
          <Grid
            container
            direction="row"
            justifyContent="center"
            alignItems="center"
            xs={12}
            marginTop={3}
          >
            {Object.entries(props.plans).filter(([key, plan]) => plan.enabled).map(([key, plan]) => {
              return <Box key={key} marginX={1}>
                <PlanCard plan={convertPlan(plan, key)} submit={undefined}/>
              </Box>;
            })}
          </Grid>
          <Grid
            item
            xs={12}
            display="flex"
            justifyContent="center"
            sx={{
              marginTop: 4,
              marginBottom: 2,
            }}
          >
            <CTA text="Get started" href="/login" />
          </Grid>
        </Grid>

        <Grid
          container
          direction="row"
          justifyContent="center"
          alignItems="center"
          marginTop={{
            xs: "40px",
            md: "60px",
          }}
        >
          <small style={{
            color: "var(--color-gray)"
          }}>© 2023 MailboxSweep, LLC</small>
        </Grid>
      </Container>
    </>
  );
}
