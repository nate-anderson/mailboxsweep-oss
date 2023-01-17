import { useEffect, useState } from "react";
import { API } from "../../client";
import { ClassifierInfo } from "../../client-gen";
import Navbar from "../../components/navbar";
import Container from '@mui/material/Container';
import CssBaseline from "@mui/material/CssBaseline";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Card from "@mui/material/Card";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import Checkbox from "@mui/material/Checkbox";
import ListItemText from "@mui/material/ListItemText";
import IconButton from "@mui/material/IconButton";
import ExpandLess from "@mui/icons-material/ExpandLess";
import ExpandMore from "@mui/icons-material/ExpandMore";
import Collapse from "@mui/material/Collapse";
import DragHandle from "@mui/icons-material/DragHandle";

const renderClassifierListItem = (classifier: ClassifierInfo, expanded: boolean, onClick: (typeKey: string) => void) => {
    
    return <>
        <ListItem
            draggable
            disablePadding
            key={classifier.typeKey}
            secondaryAction={
                <IconButton onClick={() => onClick(classifier.typeKey)}>
                    {expanded ? <ExpandLess /> : <ExpandMore />}
                </IconButton>
            }
        >
            <ListItemButton>
                
                <DragHandle sx={{
                    color: "#a3a3a3"
                }}/>
                
                <ListItemIcon>
                    <Checkbox />
                </ListItemIcon>
                <ListItemText>{classifier.name}</ListItemText>
            </ListItemButton>
        </ListItem>
        
        <Collapse in={expanded}>
            <Typography
                sx={{
                    padding: 3,
                    backgroundColor: "#ededf0"
                }}
            >
                {classifier.description}
            </Typography>
        </Collapse>
    </>
};

export default function Configure() {
    const [ classifiers, setClassifiers ] = useState<ClassifierInfo[] | undefined>();
    const [ expandedItem, setExpandedItem ] = useState<string | undefined>();
    const [ error, setError ] = useState<string | undefined>();

    useEffect(() => {
        API.getClassifiers()
            .then((res) => {
                setClassifiers(res.classifiers);
                console.log(res.classifiers);
            })
            .catch((err) => {
                setError(err);
            })
    }, []);

    const toggleExpanded = (typeKey: string) => {
        setExpandedItem(typeKey === expandedItem ? undefined : typeKey);
    }

    return (
        <>
            <Navbar />
            <Container component="main">
                <CssBaseline />
                <Box sx={{
                    marginTop: 6,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center'
                }}>
                    <Typography sx={{
                        color: "var(--color-accent)",
                        fontSize: "2em",
                        fontWeight: 100,
                    }}>
                        Let's organize your inbox.
                    </Typography>
                    <Typography
                        align="center"
                        sx={{
                            color: "var(--color-black)",
                            marginY: 2,
                            maxWidth: 600,
                            marginX: 2,
                        }}    
                    >
                        We'll put all your messages into groups based on who sent them, when they were sent, and what's inside. The recommended settings are below, but we'll organize your inbox however you like.
                    </Typography>
                </Box>
                <Card
                    sx={{
                        maxWidth: 600,
                        width: "auto",
                        marginTop: 3,
                        marginX: "auto",
                    }}
                >
                    <List>
                    {classifiers?.map(cl => renderClassifierListItem(cl, cl.typeKey === expandedItem, toggleExpanded))}
                    </List>
                </Card>
            </Container>
        </>
    )
}