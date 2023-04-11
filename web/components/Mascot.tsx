import Image from "next/image";
import MascotLogo from "../public/Mascot.svg";

export default function Mascot() {
    return (
        <Image
          src={MascotLogo}
          alt="MailboxSweep Mascot, a man dressed as a mail carrier, carrying a vintage chimney brush"
          priority
          style={{
            display: "block",
            width: "40vw",
            height: "auto",
            maxWidth: "200px",
            minWidth: "150px",
            marginBottom: "10px",
          }}
        />
      );
}