import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import Chip from "@mui/material/Chip";
import Checkbox from "@mui/material/Checkbox";
import IconChevronRight from "@mui/icons-material/ChevronRight";
import IconButton from "@mui/material/IconButton";
import { ClassificationTree } from "../../lib/tree";

export interface LeafProps {
  id: string;
  path: string[];
  label: string;
  msgCount: number;
  unreadCount: number;
  onClickExpand: (node: string) => void;
  childCount: number;
  appendSelectedNode: (path: string[]) => void;
  removeSelectedNode: (path: string[]) => void;
  onCheck: (checked: boolean) => void;
  isChecked: boolean;
  isChildChecked: boolean;
  classification: ClassificationTree | undefined;
}

export default function Leaf(props: LeafProps) {
  const toggleSelected = (selected: boolean) => {
    console.log("toggling", selected);
    if (!props.classification) {
      throw new Error("classification tree is undefined in leaf");
    }

    if (props.isChecked) {
      props.removeSelectedNode(props.path);
    } else {
      props.appendSelectedNode(props.path);
    }
  };
  return (
    <Box
      width="95vw"
      maxWidth="600px"
      sx={{
        paddingY: 1.15,
        paddingX: 1.5,
        marginTop: 1,
        borderRadius: "5px",
      }}
    >
      <Box display="flex" maxWidth="100%">
        <Checkbox
          sx={{
            paddingX: 0,
          }}
          onChange={(_, checked) => {
            toggleSelected(checked);
          }}
          checked={props.isChecked}
          indeterminate={props.isChildChecked}
        />

        <Stack
          sx={{
            alignItems: "start",
            maxWidth: "100%",
            width: "auto",
            flexGrow: "1",
            flexBasis: "100px",
            marginX: 2,
            overflowX: "hidden",
            textOverflow: "ellipsis",
          }}
          spacing={0.2}
        >
          <Typography
            sx={{
              color: "var(--color-black)",
              fontSize: "1.2em",
              overflowX: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
              width: "100%",
            }}
            style={{
              marginTop: "-6px",
            }}
          >
            {props.label}
          </Typography>
          <span
            style={{
              display: "flex",
            }}
          >
            <Chip
              sx={{
                bgcolor: "var(--color-gray)",
                color: "white",
                fontSize: "0.75em",
              }}
              label={`${props.msgCount} messages`}
              size="small"
            />
            <Chip
              sx={{
                bgcolor: "var(--color-accent)",
                marginLeft: "4px",
                fontSize: "0.75em",
                color: "white",
              }}
              label={`${props.unreadCount} unread`}
              size="small"
            />
          </span>
        </Stack>
        {props.childCount > 0 ? (
          <IconButton onClick={() => props.onClickExpand(props.id)}>
            <IconChevronRight />
          </IconButton>
        ) : undefined}
      </Box>
    </Box>
  );
}
