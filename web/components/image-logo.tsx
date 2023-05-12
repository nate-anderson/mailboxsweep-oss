import Image from "next/image";
import LogoSvg from "../public/Logo_with_text_vec.svg";

export default function ImageLogo() {
  return (
    <Image
      src={LogoSvg}
      alt="MailboxSweep Logo"
      priority
      style={{
        width: "80vw",
        height: "auto",
        maxWidth: "340px",
        marginBottom: "10px",
      }}
    />
  );
}
