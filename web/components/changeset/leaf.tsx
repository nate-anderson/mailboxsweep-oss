import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import Chip from "@mui/material/Chip";
import Checkbox from "@mui/material/Checkbox";
import IconChevronRight from "@mui/icons-material/ChevronRight";
import IconButton from "@mui/material/IconButton";

export interface LeafProps {
    id: string;
    label: string;
    msgCount: number;
    unreadCount: number;
    onClickExpand: (node: string) => void,
    children: LeafProps[];
    appendSelectedNode: (node: string) => void;
    removeSelectedNode: (node: string) => void;
}

export default function Leaf(props: LeafProps) {
    return (
        <Box
            width="95vw"
            maxWidth="600px"
            sx={{
                paddingY: 1.5,
                paddingX: 1.5,
                marginTop: 1,
                borderRadius: "5px",
            }}
        >
            <Box
                display="flex"
                justifyContent="space-between"
            >
                <Checkbox
                sx={{
                    paddingX: 0,
                }}
                />
                
                <Box
                    sx={{
                        width: "100%",
                        display: "flex",
                        fontWeight: "300",
                        marginX: 2,
                    }}
                >
                    <Stack sx={{
                        alignItems: "start",
                        maxWidth: "90%",
                        width: "auto"
                    }}>
                        <Typography
                            sx={{
                                color: "var(--color-black)",
                                fontSize: "1.2em",
                                maxWidth: "100%",
                                overflowX: "hidden",
                                textOverflow: "ellipsis",
                            }}
                            style={{
                                marginTop: "-6px",
                            }}
                        >
                            {props.label}
                        </Typography>
                        <span style={{
                            display: "flex",
                        }}>
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
                </Box>
                {
                    props.children.length > 0
                        ? <IconButton onClick={() => props.onClickExpand(props.id)}>
                            <IconChevronRight/>
                        </IconButton>
                        : undefined
                } 
            </Box>
        </Box>
    );
}