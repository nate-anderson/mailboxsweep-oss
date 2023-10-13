import Toolbar from "@mui/material/Toolbar";
import IconBack from "@mui/icons-material/ArrowBackIos";
import Typography from "@mui/material/Typography";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";

interface CrumbsProps {
  onClickBack: () => void;
  currentPath: string[];
  title?: string;
  typeKey?: string;
}

const displayTypeKey = (typeKey?: string): string => {
  switch (typeKey) {
    case "SENDER_DOMAIN":
      return "From Website";
    case "SENDER_USERNAME":
      return "From Sender";
    case "SEND_DATE":
      return "Send Date";
    case undefined:
      return "";
    default:
      return (
        typeKey
          // tokenize
          .split("_")
          // lowercase all letters
          .map((word) => word.toLowerCase())
          // capitalize first letter
          .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
          .join(" ")
      );
  }
};

function BackHomeBar(props: CrumbsProps) {
  return (
    <Toolbar
      sx={{
        maxWidth: 600,
        width: "100%",
        paddingX: 3,
      }}
      disableGutters
    >
      <IconButton
        color="primary"
        onClick={props.onClickBack}
        sx={{
          marginRight: 0,
        }}
        aria-label="Go back one level"
      >
        <IconBack />
      </IconButton>
      <Stack
        sx={{
          maxWidth: "90%",
          width: "auto",
        }}
      >
        <Typography
          fontSize=".7em"
          fontWeight="bold"
          color="var(--color-black)"
          marginY={0}
        >
          {displayTypeKey(props.typeKey)}
        </Typography>
        <Typography
          fontSize="1.2em"
          color="var(--color-gray)"
          marginTop={-0.6}
          sx={{
            overflowWrap: "break-word",
          }}
        >
          {props.title}
        </Typography>
      </Stack>
    </Toolbar>
  );
}

export default function Crumbs(props: CrumbsProps) {
  return props.currentPath.length == 0 ? (
    <Typography
      sx={{
        color: "var(--color-black)",
        fontSize: "2em",
        fontWeight: 100,
        marginBottom: 2,
        marginTop: 2,
      }}
    >
      Select Messages
    </Typography>
  ) : (
    <BackHomeBar {...props} />
  );
}
