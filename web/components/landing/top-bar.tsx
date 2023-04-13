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
import IconQuestions from "@mui/icons-material/Quiz";
import IconMarkEmailUnread from "@mui/icons-material/MarkEmailUnread";
import IconTicket from "@mui/icons-material/LocalActivity";
import IconClose from "@mui/icons-material/Close";
import IconLogin from "@mui/icons-material/Login";
import Link from "next/link";
import { useScrollTrigger } from "@mui/material";

const renderDrawerItem = (props: {
  text: string;
  icon: any;
  path: string;
  selected: boolean;
}) => (
  <Link
    key={props.text}
    href={props.path}
    passHref
    style={{ textDecoration: "none", color: "black" }}
  >
    <ListItem
      sx={{
        marginTop: 0,
        marginRight: 5,
        color: "inherit",
      }}
    >
      <ListItemButton selected={props.selected}>
        <ListItemIcon sx={{ marginLeft: 0 }}>{props.icon}</ListItemIcon>
        <ListItemText>{props.text}</ListItemText>
      </ListItemButton>
    </ListItem>
  </Link>
);

const drawerItems = [
    {
        text: "How it Works",
        icon: <IconMarkEmailUnread />,
        path: "/landing/how",
    },
    {
        text: "Plans",
        icon: <IconTicket />,
        path: "/landing/plans",
    },
    { text: "Common Questions", icon: <IconQuestions />, path: "/landing/plans" },
];

interface NavbarProps {
  page?: string;
  shadow: boolean;
}

export default function TopBar(props: NavbarProps) {
  const [drawerOpen, setDrawerOpen] = useState(false);

  const scrollTrigger = useScrollTrigger({
    disableHysteresis: true,
    threshold: 0,
  });
  
  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar
        sx={{
          backgroundColor: scrollTrigger ? "var(--color-primary)" : "var(--color-light)",
        }}
        position="fixed"
        elevation={scrollTrigger ? 4 : 0}
      >
        <Container maxWidth={false}>
          <ToolBar
            disableGutters
            sx={{
              justifyContent: "space-between",
            }}
          >
            <div style={{ width: "24px" }}></div>
            <Typography
              sx={{
                fontFamily: "Lobster",
                fontSize: "1.75em",
                color: scrollTrigger ? "white" : "var(--color-primary)",
                marginX: "auto",
              }}
            >
              MailboxSweep
            </Typography>
            <IconButton
              edge="start"
              aria-label="menu"
              onClick={() => setDrawerOpen(!drawerOpen)}
            >
              <IconMenu
                style={{
                    fill: scrollTrigger ? "white" : "var(--color-primary)"
                }}
              />
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
              justifyContent: "right",
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
              <IconClose
                style={{
                    fill: "var(--color-primary)"
                }}
              />
            </IconButton>
          </Box>
          <Link
            href="/login"
            passHref
            style={{ textDecoration: "none", color: "black" }}
        >
            <ListItem
                sx={{
                    marginTop: 2,
                    paddingY: 1,
                    marginRight: 5,
                    backgroundColor: "var(--color-accent)",
                    color: "white",
                }}
            >
            <ListItemButton>
                <ListItemIcon sx={{ marginLeft: 0 }}><IconLogin style={{ fill: "white" }}/></ListItemIcon>
                <ListItemText>Log In or Sign Up</ListItemText>
            </ListItemButton>
            </ListItem>
           
          </Link>
          {drawerItems.map((item) =>
            renderDrawerItem({
              ...item,
              selected: props.page === item.text,
            })
          )}
        </Drawer>
      </AppBar>
      <ToolBar />
    </Box>
  );
}
