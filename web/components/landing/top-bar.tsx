import AppBar from "@mui/material/AppBar";
import ToolBar from "@mui/material/Toolbar";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Container from "@mui/material/Container";
import IconButton from "@mui/material/IconButton";
import IconMenu from "@mui/icons-material/Menu";
import Drawer from "@mui/material/Drawer";
import { MutableRefObject, useState } from "react";
import ListItem from "@mui/material/ListItem";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import IconClose from "@mui/icons-material/Close";
import IconLogin from "@mui/icons-material/Login";
import Link from "next/link";
import { useScrollTrigger } from "@mui/material";

type Action =  {
  path?: string;
  scrollRef?: MutableRefObject<HTMLElement | null>;
};

const smoothScrollToRef = (ref: MutableRefObject<HTMLElement | null>, margin: number = 0) => {
  const dims = ref.current!.getBoundingClientRect();
  const top = dims.top + window.pageYOffset - margin;
  window.scrollTo({ behavior: "smooth", top });
}

function DrawerItem(props: {
  text: string;
  icon: any;
  action: Action;
  selected: boolean;
  closeDrawer: () => void;
  currentlyScrolledRef?: MutableRefObject<HTMLElement | null>;
  setCurrentlyScrolledRef?: (ref: MutableRefObject<HTMLElement | null>) => void;
}) {
  return <ListItem
      key={props.text}
      onClick={
          props.action.scrollRef
            ? () => {
              smoothScrollToRef(props.action.scrollRef!, 100);
              props.setCurrentlyScrolledRef!(props.action.scrollRef!);
              props.closeDrawer();
            }
            : () => { window.location.href = props.action.path ?? "" }
      }
      style={{ textDecoration: "none", color: "black" }}
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
    </ListItem>;
};

interface DrawerItem {
  text: string;
  icon: JSX.Element;
  // either link to another page, or scroll to a ref
  action: Action;
}

interface NavbarProps {
  page?: string;
  shadow: boolean;
  drawerItems: DrawerItem[];
}

export default function TopBar(props: NavbarProps) {
  const [drawerOpen, setDrawerOpen] = useState(false);

  const [ currentlyScrolledRef, setCurrentlyScrolledRef ] = useState<MutableRefObject<HTMLElement | null>>(); 

  // decides the color/transparency of the top bar based on scroll position
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
          {props.drawerItems.map((item) =>
            <DrawerItem
              key={item.text}
              text={item.text}
              action={item.action}
              icon={item.icon}
              selected={false}
              closeDrawer={() => setDrawerOpen(false)}
              currentlyScrolledRef={currentlyScrolledRef}
              setCurrentlyScrolledRef={setCurrentlyScrolledRef}
            />
          )}
        </Drawer>
      </AppBar>
      <ToolBar />
    </Box>
  );
}
