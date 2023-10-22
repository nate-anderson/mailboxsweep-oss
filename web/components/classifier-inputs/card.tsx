import { SelectedClassifier } from "../../lib/classifier";
import Card from "@mui/material/Card";
import ListItemButton from "@mui/material/ListItemButton";
import IconButton from "@mui/material/IconButton";
import ExpandLess from "@mui/icons-material/ExpandLess";
import ExpandMore from "@mui/icons-material/ExpandMore";
import Collapse from "@mui/material/Collapse";
import DragHandle from "@mui/icons-material/DragHandle";
import Typography from "@mui/material/Typography";
import { ReactNode } from "react";
import IconRemove from "@mui/icons-material/DeleteForever";
import {
  DraggableProvidedDraggableProps,
  DraggableProvidedDragHandleProps,
} from "react-beautiful-dnd";

interface Props {
  classifier: SelectedClassifier;
  expanded: boolean;
  toggleExpanded: () => void;
  children?: ReactNode;
  remove: () => void;
  draggableRef: (element: HTMLElement | null) => void;
  draggableProps: DraggableProvidedDraggableProps;
  dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
  canRemove: boolean;
}

export default function ClassifierCard(props: Props) {
  return (
    <Card
      data-testid={`classifier-card-${props.classifier.typeKey}`}
      sx={{
        marginBottom: 1,
      }}
      ref={props.draggableRef}
      {...props.draggableProps}
    >
      <ListItemButton
        sx={{
          paddingY: 2,
        }}
        {...props.dragHandleProps}
      >
        {
          props.expanded && props.canRemove
            ? <IconButton
              sx={{
                marginLeft: -1,
                marginRight: 1,
              }}
              onClick={() => props.remove()}
            >
              <IconRemove color="error" />
            </IconButton>
            : <DragHandle
            data-testid={`classifier-drag-handle-${props.classifier.typeKey}`}
            sx={{
              color: "var(--color-accent)",
              marginRight: 2,
            }}
          />
        }
        
        {props.children}

        <IconButton data-testid={`classifier-expand-toggle-${props.classifier.typeKey}`} onClick={() => props.toggleExpanded()}>
          {props.expanded ? <ExpandLess /> : <ExpandMore />}
        </IconButton>
      </ListItemButton>
      <Collapse in={props.expanded}>
        <Typography
          sx={{
            padding: 2,
          }}
        >
          {props.classifier.description}
        </Typography>
      </Collapse>
    </Card>
  );
}
