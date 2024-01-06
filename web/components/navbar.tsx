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
import IconEnvelope from "@mui/icons-material/Email";
import IconCreditCard from "@mui/icons-material/CreditCard";
import IconClose from "@mui/icons-material/Close";
import IconHome from "@mui/icons-material/Home";
import Link from "next/link";
import IconLogout from "@mui/icons-material/Logout";
import { useAPIClient } from "../client";
import { useRouter } from "next/router";
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import Button from "@mui/material/Button";
import FormControlLabel from "@mui/material/FormControlLabel";
import Checkbox from "@mui/material/Checkbox";


const renderDrawerItem = (props: {
  text: string;
  icon: any;
  path?: string;
  action?: () => void;
  selected: boolean;
}) => props.path ? (
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
        <ListItemIcon sx={{ marginLeft: 0 }} data-testid={`menu-item-${props.text.toLowerCase()}`}>{props.icon}</ListItemIcon>
        <ListItemText>{props.text}</ListItemText>
      </ListItemButton>
    </ListItem>
  </Link>
)
: (
  <Link
    href="#"
    key={props.text}
    onClick={props.action}
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

interface DrawerItem {
  text: string;
  icon: JSX.Element;
  path?: string;
  action?: () => void;
}

interface NavbarProps {
  page?: string;
}

export default function Navbar(props: NavbarProps) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [logoutModalOpen, setLogoutModalOpen] = useState(false);
  const [logoutAllDevices, setLogoutAllDevices] = useState(true);
  const API = useAPIClient();
  const router = useRouter();

  const doLogout = () => {
    API.logout(logoutAllDevices ? 1 : 0)
      .then(() => router.push("/"))
      .catch((err) => console.log("Logout failed, please try again momentarily."));
  }

  const drawerItems: DrawerItem[] = [
    { text: "Home", icon: <IconHome />, path: "/home" },
    {
      text: "Email Account",
      icon: <IconEnvelope />,
      path: "/email-account/login",
    },
    { text: "Payment & Plan", icon: <IconCreditCard />, path: "/plan" },
    { text: "Logout", icon: <IconLogout />, action: () => setLogoutModalOpen(true) }
  ];

  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar
        sx={{
          backgroundColor: "var(--color-primary)",
        }}
        position="static"
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
              variant="h1"
              sx={{
                fontFamily: "Lobster",
                fontSize: "1.5em",
                color: "white",
                marginX: "auto",
              }}
            >
              MailboxSweep
            </Typography>
            <IconButton
              id="sidebar-menu-open"
              edge="start"
              color="inherit"
              aria-label="menu"
              onClick={() => setDrawerOpen(!drawerOpen)}
              data-testid="sidebar-menu-button"
            >
              <IconMenu />
            </IconButton>
          </ToolBar>
        </Container>
        <Drawer
          id="menu-drawer"
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
              id="navbar-menu-close"
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
          {drawerItems.map((item) =>
            renderDrawerItem({
              ...item,
              selected: props.page === item.text,
            })
          )}
        </Drawer>
      </AppBar>
      <Dialog
        open={logoutModalOpen}
        onClose={() => setLogoutModalOpen(false)}
        aria-labelledby="logout-dialog-title"
        aria-describedby="logout-dialog-description"
      >
        <DialogTitle id="logout-dialog-title">
          Log out of MailboxSweep
        </DialogTitle>
        <DialogContent>
          <DialogContentText id="logout-dialog-description">
            You'll need to sign in again to use the app. You can also log out on any other devices you've used.
          </DialogContentText>
        </DialogContent>
        <Box
          marginY={0}
          display="flex"
          justifyContent="end"
        >
          <FormControlLabel
            labelPlacement="start"
            
            control={<Checkbox
              sx={{
                paddingY: 0,
              }}
              checked={logoutAllDevices}
              onChange={(_, checked) => setLogoutAllDevices(checked)}
              data-testid="logout-all-checkbox"
            />}
            label="Log all of my devices out"
            sx={{
              marginRight: 1,
              color: "var(--color-gray)"
            }}
          />
        </Box>
        <DialogActions
          sx={{
            margin: 1,
          }}
        >
          <Button onClick={() => doLogout()} autoFocus startIcon={<IconLogout />} data-testid="logout-confirm-button">
            Logout
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
