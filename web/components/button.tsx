import { SxProps, Theme } from "@mui/material";
import MUIButton, { ButtonProps as MUIButtonProps } from "@mui/material/Button";

type ButtonProps = MUIButtonProps & {
  children?: string;
  sx?: SxProps<Theme> | undefined;
  disabled?: boolean;
  onClick: () => any;
};

export default function Button(props: ButtonProps) {
  return (
    <MUIButton
      variant="contained"
      size="large"
      sx={{
        backgroundColor: "var(--color-accent)",
        color: "white",
        ...props.sx,
      }}
      disabled={props.disabled ?? false}
      {...props}
    >
      {props.children}
    </MUIButton>
  );
}
