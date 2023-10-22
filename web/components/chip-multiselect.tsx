import Box from "@mui/material/Box";
import Chip from "@mui/material/Chip";
import IconCheckCircleOutline from "@mui/icons-material/CheckCircleOutline";
import IconCircle from "@mui/icons-material/RadioButtonUnchecked";

interface ChipMultiselectProps {
    items: Set<string>;
    selectedItems: Set<string>;
    onAdd: (add: string) => void;
    onRemove: (remove: string) => void;
}

function SelectChip(props: {selected: boolean, label: string, onRemove: (key: string) => void, onAdd: (key:string) => void}) {
    return <Chip
        icon={props.selected ? <IconCheckCircleOutline /> : <IconCircle />}
        size="small"
        sx={{
            marginLeft: 1,
            marginTop: 1,
        }}
        color={props.selected ? "primary" : "default"}
        key={props.label}
        label={props.label}
        onClick={() => props.selected ? props.onRemove(props.label) : props.onAdd(props.label)}
        data-selected={props.selected.toString()}
        data-testid={`select-chip-folder-${props.label}`}
    />
}

export default function ChipMultiselect(props: ChipMultiselectProps) {
    return <Box
        display="flex"
        justifyContent="left"
        flexWrap="wrap"
        sx={{
            listStyle: "none",
            marginBottom: 1,
        }}
    >
        {
        [...props.items].map(str => {
            const selected = props.selectedItems.has(str);
            return <SelectChip
                selected={selected}
                key={str}
                label={str}
                onAdd={props.onAdd}
                onRemove={props.onRemove}
            />;
        })
        }
    </Box>
}