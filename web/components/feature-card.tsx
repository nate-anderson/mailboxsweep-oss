import { SxProps, Theme, Typography } from "@mui/material";
import Card from "@mui/material/Card";
import Box from "@mui/material/Box";
import IconShieldOutline from "@mui/icons-material/ShieldOutlined";
import IconLockOutline from "@mui/icons-material/LockOutlined";
import IconFenceOutline from '@mui/icons-material/FenceOutlined';

type IconType = "shield" | "lock" | "fence";

interface FeatureCardProps {
    headline: string;
    icon: IconType;
    description: string;
}

const iconStyle: SxProps<Theme> = {
    fontSize: "3em",
    color: "var(--color-accent)",
    marginTop: 1,
    marginBottom: 2,
};

const icons: Record<IconType, JSX.Element> = {
    "shield": <IconShieldOutline sx={iconStyle} />,
    "lock": <IconLockOutline sx={iconStyle} />,
    "fence": <IconFenceOutline sx={iconStyle} />,
}

export default function FeatureCard(props: FeatureCardProps) {
    return <Box
        sx={{
            // backgroundColor: "rgba(135, 135, 135, 0.1)",
            maxWidth: 200,
            width: "auto",
            marginX: 2,
            marginBottom: 2,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            minHeight: "200px"
        }}
    >
        {icons[props.icon]}
        <Typography
            fontWeight="bold"
            align="center"
            marginY={1}
            fontSize="1.2em"
        >{props.headline}</Typography>
        <Typography
            align="center"
        >{props.description}</Typography>
    </Box>
}