import Box from "@mui/material/Box";
import Link from "next/link";
import Button from "@mui/material/Button";
import IconChevronRight from "@mui/icons-material/ChevronRight";

export default function ActionLink(props: {
    color: string;
    backgroundColor: string;
    href: string;
    text: string;
  }) {
    return (
      <Link
        href={props.href}
        style={{
          textDecoration: "none",
          width: "100%",
        }}
      >
        <Box display="flex" justifyContent="end">
          <Button
            endIcon={<IconChevronRight />}
            size="large"
            sx={{
              color: props.color,
              border: `1px solid ${props.backgroundColor}`,
              backgroundColor: props.backgroundColor,
              fontWeight: "bold",
              marginRight: 0,
              marginLeft: "auto",
              "&:hover": {
                color: props.backgroundColor,
                backgroundColor: props.color,
                borderColor: props.color,
              },
            }}
          >
            {props.text}
          </Button>
        </Box>
      </Link>
    );
  }