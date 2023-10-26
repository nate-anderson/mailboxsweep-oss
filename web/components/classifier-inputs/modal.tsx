import Dialog from "@mui/material/Dialog";
import DialogTitle from "@mui/material/DialogTitle";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemText from "@mui/material/ListItemText";
import ListItemButton from "@mui/material/ListItemButton";
import IconPlusBox from "@mui/icons-material/AddBox";
import { ClassifierInfo } from "../../client-gen";
import IconButton from "@mui/material/IconButton";
import IconPerson from "@mui/icons-material/Person";
import IconGlobe from "@mui/icons-material/Public";
import IconCalendar from "@mui/icons-material/CalendarToday";
import IconTitle from "@mui/icons-material/Title";
import IconFilter from "@mui/icons-material/FilterAlt";
import Divider from '@mui/material/Divider';

interface DialogOptionProps {
    onClose: () => void;
    addClassifier: (cl: ClassifierInfo) => void;
    classifier: ClassifierInfo;
}

const Icons: Record<string, JSX.Element> = {
    "SENDER_DOMAIN": <IconGlobe />,
    "SENDER_USERNAME": <IconPerson />,
    "SEND_DATE": <IconCalendar />,
    "SUBJECT_KEYWORD": <IconTitle />,
};

const defaultIcon = <IconFilter />;

const classifierIcon = (typeKey: string) => {
    if (Icons.hasOwnProperty(typeKey)) {
        return Icons[typeKey];
    }
    return defaultIcon;
}

export function DialogOption(props: DialogOptionProps) {
    return <div key={props.classifier.typeKey}>
        <ListItem key={props.classifier.typeKey}
            sx={{
                paddingLeft: 0,
                paddingTop: 0,
            }}
            secondaryAction={
                <IconButton
                    sx={{
                        marginRight: 1,
                        paddingX: 0,
                    }}
                    onClick={() => {
                        props.addClassifier(props.classifier);
                        props.onClose()
                        }
                    }
                    aria-label={`Add classifier ${props.classifier.name} to custom configuration`}
                >
                    <IconPlusBox color="primary" />
                </IconButton>
            }
            >
            <ListItemButton
                disableRipple
                disableTouchRipple
                disableGutters
                sx={{
                    paddingRight: "14px !important",
                    paddingLeft: 1,
                    "&:hover": {
                        backgroundColor: "transparent",
                    },
                }}
            >
                <IconButton>
                    {classifierIcon(props.classifier.typeKey)}
                </IconButton>
                    
                <ListItemText
                    primary={props.classifier.name}
                    secondary={props.classifier.description}
                    sx={{
                        marginLeft: 1,
                    }}
                />
            </ListItemButton>
        </ListItem>
    </div>;
}

interface AddClassifierDialogProps {
    open: boolean;
    onClose: () => void;
    availableClassifiers: ClassifierInfo[];
    addClassifier: (cl: ClassifierInfo) => void;
}

export function AddClassifierDialog(props: AddClassifierDialogProps) {
    return <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Add Classification Filter</DialogTitle>
      <List
        sx={{ pt: 0, mb: 1 }}
      >
        {props.availableClassifiers.map((cl, i) => {
          return <div key={cl.typeKey}>
            <DialogOption
                onClose={props.onClose}
                addClassifier={props.addClassifier}
                classifier={cl}
            />
            {
                i !== (props.availableClassifiers.length - 1)
                    ? <Divider variant="inset" component="li" sx={{marginLeft: 0}} />
                    : <></>
            }
          </div>
        })}
      </List>
    </Dialog>
  }