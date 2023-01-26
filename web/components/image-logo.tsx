import { Box } from "@mui/material";
import Image from "next/image";
import LogoSvg from "../public/Logo_with_text.svg";
import LargeLogo from "./large-logo";

export default function ImageLogo() {
    return (<Image
            src={LogoSvg}
            alt="MailboxSweep Logo"
            priority
            style={{
                width: "80vw",
                height: "auto",
                maxWidth: "340px",
                marginBottom: "10px",
            }}
        />);
}